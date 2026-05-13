package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the budget limiter server.
type Config struct {
	// Server configuration
	GRPCPort    int `envconfig:"GRPC_PORT" default:"4444"`
	HTTPPort    int `envconfig:"HTTP_PORT" default:"8080"`
	MetricsPort int `envconfig:"METRICS_PORT" default:"9090"`

	// Database configuration
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// Cache configuration
	ModelCostCacheTTL time.Duration `envconfig:"MODEL_COST_CACHE_TTL" default:"60s"`
	BudgetCacheTTL    time.Duration `envconfig:"BUDGET_CACHE_TTL" default:"30s"`

	// Reservation configuration
	ReservationTTL     time.Duration `envconfig:"RESERVATION_TTL" default:"5m"`
	ReservationCleanup time.Duration `envconfig:"RESERVATION_CLEANUP" default:"1m"`

	// Period reset check interval
	PeriodResetInterval time.Duration `envconfig:"PERIOD_RESET_INTERVAL" default:"1m"`

	// Alert worker check interval
	AlertCheckInterval time.Duration `envconfig:"ALERT_CHECK_INTERVAL" default:"60s"`

	// Days-until-exhausted threshold for forecast_exhaustion alerts
	ForecastAlertDaysThreshold float64 `envconfig:"FORECAST_ALERT_DAYS_THRESHOLD" default:"3"`

	// Default cost estimation multiplier (for pre-request budget checks)
	DefaultEstimationMultiplier float64 `envconfig:"DEFAULT_ESTIMATION_MULTIPLIER" default:"1.5"`

	// Default estimated tokens per request (when we can't parse the request)
	DefaultEstimatedInputTokens  int64 `envconfig:"DEFAULT_ESTIMATED_INPUT_TOKENS" default:"1000"`
	DefaultEstimatedOutputTokens int64 `envconfig:"DEFAULT_ESTIMATED_OUTPUT_TOKENS" default:"1000"`

	// Log level
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Header names for entity identification (Option A from RFE)
	OrgIDHeader  string `envconfig:"ORG_ID_HEADER" default:"x-gw-org-id"`
	TeamIDHeader string `envconfig:"TEAM_ID_HEADER" default:"x-gw-team-id"`
	UserIDHeader string `envconfig:"USER_ID_HEADER" default:"x-user-id"`
	ModelHeader  string `envconfig:"MODEL_HEADER" default:"x-gateway-model-name"`

	// JWT claim keys for extracting identity from Authorization Bearer token
	// These are used as fallback when headers are not present
	OrgIDClaim  string `envconfig:"ORG_ID_CLAIM" default:"org_id"`
	TeamIDClaim string `envconfig:"TEAM_ID_CLAIM" default:"team_id"`

	// Audit log retention
	AuditRetentionDays int `envconfig:"AUDIT_RETENTION_DAYS" default:"90"`

	// Database pool
	DBMaxConnections int32 `envconfig:"DB_MAX_CONNECTIONS" default:"10"`

	// PostgreSQL TLS
	DBSSLMode       string `envconfig:"DB_SSL_MODE" default:"disable"`
	DBSSLCACert     string `envconfig:"DB_SSL_CA_CERT"`
	DBSSLClientCert string `envconfig:"DB_SSL_CLIENT_CERT"`
	DBSSLClientKey  string `envconfig:"DB_SSL_CLIENT_KEY"`
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
