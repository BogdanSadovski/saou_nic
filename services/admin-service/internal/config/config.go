package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	RBAC     RBACConfig     `mapstructure:"rbac"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Mode            string        `mapstructure:"mode"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

// GRPCConfig holds gRPC server configuration.
type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

// JWTConfig holds JWT authentication configuration.
type JWTConfig struct {
	Secret        string        `mapstructure:"secret"`
	TokenTTL      time.Duration `mapstructure:"token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

// RBACConfig holds role-based access control configuration.
type RBACConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}

// Load reads configuration from the given config file path.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Env-var overrides — both POSTGRES_* (compose convention) and DB_*
	// (legacy) honoured. Allows changing DB credentials/host without an
	// image rebuild.
	_ = v.BindEnv("database.host", "POSTGRES_HOST", "DB_HOST")
	_ = v.BindEnv("database.port", "POSTGRES_PORT", "DB_PORT")
	_ = v.BindEnv("database.user", "POSTGRES_USER", "DB_USER")
	_ = v.BindEnv("database.password", "POSTGRES_PASSWORD", "DB_PASSWORD")
	_ = v.BindEnv("database.dbname", "POSTGRES_DB", "DB_NAME")
	_ = v.BindEnv("jwt.secret", "JWT_SECRET")

	v.SetDefault("server.port", 8081)
	v.SetDefault("server.mode", "release")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.shutdown_timeout", "10s")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "admin")
	v.SetDefault("database.password", "admin")
	v.SetDefault("database.dbname", "admin_service")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_conns", 10)
	v.SetDefault("database.min_conns", 2)
	v.SetDefault("database.max_conn_lifetime", "1h")
	v.SetDefault("database.max_conn_idle_time", "30m")

	v.SetDefault("grpc.port", 50051)

	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.token_ttl", "24h")
	v.SetDefault("jwt.refresh_token_ttl", "720h")

	v.SetDefault("rbac.enabled", true)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
