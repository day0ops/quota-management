package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentgateway/quota-management/internal/alert"
	"github.com/agentgateway/quota-management/internal/api"
	"github.com/agentgateway/quota-management/internal/audit"
	"github.com/agentgateway/quota-management/internal/cel"
	"github.com/agentgateway/quota-management/internal/config"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/metrics"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Configure logging
	configureLogging(cfg.LogLevel)

	log.Info().
		Int("http_port", cfg.HTTPPort).
		Int("metrics_port", 9091).
		Msg("starting budget management UI server")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
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

	// Create repository
	repo := db.NewRepository(database)

	// Create CEL evaluator
	celEvaluator, err := cel.NewEvaluator()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create CEL evaluator")
	}

	// Create audit service
	auditSvc := audit.NewService(repo)

	// Create alert service
	alertSvc := alert.NewService(repo, cfg)

	// Create rate limit repository and handler
	rateLimitRepo := db.NewRateLimitRepository(database)
	rateLimitHandler := api.NewRateLimitHandler(rateLimitRepo, auditSvc)

	// Create HTTP server for management API and UI
	router := mux.NewRouter()
	apiHandler := api.NewHandler(repo, celEvaluator, auditSvc)
	apiHandler.RegisterRoutes(router)
	rateLimitHandler.RegisterRoutes(router)

	// Serve static UI files
	uiPath := "/app/ui"
	if envUIPath := os.Getenv("UI_PATH"); envUIPath != "" {
		uiPath = envUIPath
	}
	if _, err := os.Stat(uiPath); err == nil {
		// Serve static files
		fs := http.FileServer(http.Dir(uiPath))
		router.PathPrefix("/").Handler(spaHandler{staticPath: uiPath, indexPath: "index.html", fileServer: fs})
		log.Info().Str("path", uiPath).Msg("serving UI static files")
	}

	// Wrap router with CORS middleware
	corsHandler := enableCORS(router)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      corsHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start background workers
	go runBackgroundWorkers(ctx, auditSvc, alertSvc, cfg)

	// Start HTTP server
	go func() {
		log.Info().Str("addr", httpServer.Addr).Msg("starting HTTP server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Create and start metrics server on port 9091
	metricsServer := &http.Server{
		Addr:         ":9091",
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

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Info().Str("signal", sig.String()).Msg("received shutdown signal")

	// Graceful shutdown
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("metrics server shutdown error")
	}

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

	// Use console writer for development
	if os.Getenv("LOG_FORMAT") != "json" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

// enableCORS adds CORS headers to allow UI requests
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// spaHandler serves the SPA and handles client-side routing
type spaHandler struct {
	staticPath string
	indexPath  string
	fileServer http.Handler
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Check if the file exists
	fullPath := h.staticPath + path
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		// File doesn't exist, serve index.html for SPA routing
		http.ServeFile(w, r, h.staticPath+"/"+h.indexPath)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// File exists, serve it
	h.fileServer.ServeHTTP(w, r)
}

func runBackgroundWorkers(ctx context.Context, auditSvc *audit.Service, alertSvc *alert.Service, cfg *config.Config) {
	auditRetentionTicker := time.NewTicker(24 * time.Hour)
	defer auditRetentionTicker.Stop()

	alertCheckTicker := time.NewTicker(cfg.AlertCheckInterval)
	defer alertCheckTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-auditRetentionTicker.C:
			count, err := auditSvc.CleanupOldEntries(ctx, cfg.AuditRetentionDays)
			if err != nil {
				log.Error().Err(err).Msg("failed to cleanup old audit logs")
			} else if count > 0 {
				log.Info().Int64("count", count).Msg("cleaned up old audit log entries")
			}
		case <-alertCheckTicker.C:
			if err := alertSvc.CheckAndUpdateAlerts(ctx); err != nil {
				log.Error().Err(err).Msg("failed to check alerts")
			}
		}
	}
}
