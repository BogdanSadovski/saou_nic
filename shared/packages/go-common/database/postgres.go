package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
)

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	Database        string        `yaml:"database" json:"database"`
	User            string        `yaml:"user" json:"user"`
	Password        string        `yaml:"password" json:"password"`
	SSLMode         string        `yaml:"ssl_mode" json:"ssl_mode"`
	MaxConns        int32         `yaml:"max_conns" json:"max_conns"`
	MinConns        int32         `yaml:"min_conns" json:"min_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime" json:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time" json:"max_conn_idle_time"`
	ConnTimeout     time.Duration `yaml:"conn_timeout" json:"conn_timeout"`
}

// DefaultPostgresConfig returns a PostgresConfig with sensible defaults.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		Database:        "real_ass",
		User:            "postgres",
		Password:        "postgres",
		SSLMode:         "disable",
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		ConnTimeout:     10 * time.Second,
	}
}

// DSN returns the database connection string.
func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// Postgres represents a PostgreSQL database connection pool.
type Postgres struct {
	pool   *pgxpool.Pool
	config PostgresConfig
}

// NewPostgres creates a new Postgres instance without connecting.
func NewPostgres(cfg PostgresConfig) *Postgres {
	return &Postgres{config: cfg}
}

// Connect establishes a connection to PostgreSQL using the configured DSN.
func (p *Postgres) Connect(ctx context.Context) error {
	connStr := p.config.DSN()

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	poolConfig.MaxConns = p.config.MaxConns
	poolConfig.MinConns = p.config.MinConns
	poolConfig.MaxConnLifetime = p.config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = p.config.MaxConnIdleTime
	if p.config.ConnTimeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = p.config.ConnTimeout
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.pool = pool

	logger.Info("connected to PostgreSQL",
		zap.String("host", p.config.Host),
		zap.Int("port", p.config.Port),
		zap.String("database", p.config.Database),
	)

	return nil
}

// Close closes the database connection pool.
func (p *Postgres) Close() {
	if p.pool != nil {
		p.pool.Close()
		logger.Info("PostgreSQL connection pool closed")
	}
}

// Ping checks if the database connection is alive.
func (p *Postgres) Ping(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("database not connected")
	}
	return p.pool.Ping(ctx)
}

// Pool returns the underlying connection pool.
func (p *Postgres) Pool() *pgxpool.Pool {
	return p.pool
}

// HealthCheck performs a comprehensive health check on the database connection.
func (p *Postgres) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database not connected")
	}

	health := make(map[string]interface{})

	// Ping check
	if err := p.pool.Ping(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		return health, err
	}

	// Get pool statistics
	stats := p.pool.Stat()
	health["status"] = "healthy"
	health["total_conns"] = stats.TotalConns()
	health["acquired_conns"] = stats.AcquiredConns()
	health["idle_conns"] = stats.IdleConns()
	health["max_conns"] = p.config.MaxConns
	health["database"] = p.config.Database
	health["host"] = p.config.Host

	return health, nil
}

// Exec executes a query without returning any rows.
func (p *Postgres) Exec(ctx context.Context, sql string, args ...interface{}) (pgxpool.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

// Query executes a query that returns rows.
func (p *Postgres) Query(ctx context.Context, sql string, args ...interface{}) (pgxpool.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row.
func (p *Postgres) QueryRow(ctx context.Context, sql string, args ...interface{}) pgxpool.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}
