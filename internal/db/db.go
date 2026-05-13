package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgx connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// Config holds database connection configuration.
type Config struct {
	DatabaseURL    string
	MaxConnections int32
	SSLMode        string
	SSLCACert      string
	SSLClientCert  string
	SSLClientKey   string
}

// New creates a new database connection pool with optional TLS.
func New(ctx context.Context, cfg Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	if cfg.MaxConnections > 0 {
		poolConfig.MaxConns = cfg.MaxConnections
	}

	if cfg.SSLMode != "" && cfg.SSLMode != "disable" {
		tlsConfig, err := buildTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		poolConfig.ConnConfig.TLSConfig = tlsConfig
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func buildTLSConfig(cfg Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	switch cfg.SSLMode {
	case "require":
		tlsConfig.InsecureSkipVerify = true
	case "verify-ca", "verify-full":
		if cfg.SSLCACert == "" {
			return nil, fmt.Errorf("DB_SSL_CA_CERT required for ssl mode %s", cfg.SSLMode)
		}
		caCert, err := os.ReadFile(cfg.SSLCACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.RootCAs = caCertPool
		if cfg.SSLMode == "verify-full" {
			tlsConfig.InsecureSkipVerify = false
		} else {
			tlsConfig.InsecureSkipVerify = true
		}
	}

	if cfg.SSLClientCert != "" && cfg.SSLClientKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.SSLClientCert, cfg.SSLClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert/key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}
