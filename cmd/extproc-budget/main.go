package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentgateway/quota-management/internal/budget"
	"github.com/agentgateway/quota-management/internal/cel"
	"github.com/agentgateway/quota-management/internal/config"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/extproc"
	"github.com/agentgateway/quota-management/internal/metrics"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	configureLogging(cfg.LogLevel)

	log.Info().
		Int("grpc_port", cfg.GRPCPort).
		Int("metrics_port", cfg.MetricsPort).
		Msg("starting quota-budget ext-proc server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.New(ctx, db.Config{
		DatabaseURL:    cfg.DatabaseURL,
		MaxConnections: cfg.DBMaxConnections,
		SSLMode:        cfg.DBSSLMode,
		SSLCACert:      cfg.DBSSLCACert,
		SSLClientCert:  cfg.DBSSLClientCert,
		SSLClientKey:   cfg.DBSSLClientKey,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer database.Close()

	if cfg.DBSSLMode != "" && cfg.DBSSLMode != "disable" {
		log.Info().Str("db_ssl_mode", cfg.DBSSLMode).Msg("connected to database (PostgreSQL TLS)")
	} else {
		log.Info().Msg("connected to database")
	}

	repo := db.NewRepository(database)

	celEvaluator, err := cel.NewEvaluator()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create CEL evaluator")
	}

	budgetSvc := budget.NewService(repo, celEvaluator, cfg)

	if err := budgetSvc.RefreshModelCostCache(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to refresh model cost cache")
	}

	if err := budgetSvc.RefreshBudgetMetrics(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to initialize budget metrics")
	}

	extprocServer := extproc.NewBudgetServer(budgetSvc, celEvaluator, cfg)

	grpcServer := grpc.NewServer()
	extprocServer.Register(grpcServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	go runBackgroundWorkers(ctx, budgetSvc, cfg)

	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal().Err(err).Str("addr", grpcAddr).Msg("failed to listen")
	}

	go func() {
		log.Info().Str("addr", grpcAddr).Msg("starting gRPC server")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler:      metrics.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Str("addr", metricsServer.Addr).Msg("starting metrics server")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("metrics server failed")
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Info().Str("signal", sig.String()).Msg("received shutdown signal")

	cancel()

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("metrics server shutdown error")
	}

	grpcServer.GracefulStop()

	log.Info().Msg("server stopped")
}

func configureLogging(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if os.Getenv("LOG_FORMAT") != "json" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

func runBackgroundWorkers(ctx context.Context, budgetSvc *budget.Service, cfg *config.Config) {
	periodResetTicker := time.NewTicker(cfg.PeriodResetInterval)
	defer periodResetTicker.Stop()

	reservationCleanupTicker := time.NewTicker(cfg.ReservationCleanup)
	defer reservationCleanupTicker.Stop()

	modelCostRefreshTicker := time.NewTicker(cfg.ModelCostCacheTTL)
	defer modelCostRefreshTicker.Stop()

	budgetMetricsTicker := time.NewTicker(30 * time.Second)
	defer budgetMetricsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-periodResetTicker.C:
			count, err := budgetSvc.ResetExpiredBudgets(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to reset expired budgets")
			} else if count > 0 {
				log.Info().Int64("count", count).Msg("reset expired budgets")
			}
		case <-reservationCleanupTicker.C:
			count, err := budgetSvc.CleanupExpiredReservations(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to cleanup expired reservations")
			} else if count > 0 {
				log.Info().Int64("count", count).Msg("cleaned up expired reservations")
			}
		case <-modelCostRefreshTicker.C:
			if err := budgetSvc.RefreshModelCostCache(ctx); err != nil {
				log.Error().Err(err).Msg("failed to refresh model cost cache")
			}
		case <-budgetMetricsTicker.C:
			if err := budgetSvc.RefreshBudgetMetrics(ctx); err != nil {
				log.Error().Err(err).Msg("failed to refresh budget metrics")
			}
		}
	}
}
