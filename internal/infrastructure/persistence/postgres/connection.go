package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds PostgreSQL connection configuration.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Host:            "localhost",
		Port:            5432,
		User:            "bridge",
		Password:        "bridge",
		Database:        "bridge",
		SSLMode:         "disable",
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// ConnectionString returns the PostgreSQL connection string.
func (c *Config) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// Pool wraps a pgxpool.Pool with logging.
type Pool struct {
	pool   *pgxpool.Pool
	logger *bolt.Logger
}

// NewPool creates a new connection pool.
func NewPool(ctx context.Context, cfg *Config, logger *bolt.Logger) (*Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Int("max_conns", int(cfg.MaxConns)).
		Msg("PostgreSQL connection pool created")

	return &Pool{pool: pool, logger: logger}, nil
}

// Pool returns the underlying pgxpool.Pool.
func (p *Pool) Pool() *pgxpool.Pool {
	return p.pool
}

// Close closes the connection pool.
func (p *Pool) Close() {
	p.pool.Close()
	p.logger.Info().Msg("PostgreSQL connection pool closed")
}

// Ping verifies the connection.
func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Stats returns pool statistics.
func (p *Pool) Stats() *pgxpool.Stat {
	return p.pool.Stat()
}

// HealthCheck returns true if the connection is healthy.
func (p *Pool) HealthCheck(ctx context.Context) bool {
	return p.Ping(ctx) == nil
}
