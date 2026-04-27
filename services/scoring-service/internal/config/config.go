package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration loaded from YAML and environment.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	GRPC     GRPCConfig     `yaml:"grpc"`
	Logging  LoggingConfig  `yaml:"logging"`
	Scoring  ScoringConfig  `yaml:"scoring"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// DatabaseConfig contains PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	DBName          string `yaml:"dbname"`
	SSLMode         string `yaml:"ssl_mode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

// GRPCConfig contains gRPC server settings.
type GRPCConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// ScoringConfig contains scoring-specific settings.
type ScoringConfig struct {
	DefaultPassThreshold float64         `yaml:"default_pass_threshold"`
	EvaluationTimeout    time.Duration   `yaml:"evaluation_timeout"`
	MaxConcurrentEvals   int             `yaml:"max_concurrent_evals"`
	Weights              ScoringWeights  `yaml:"weights"`
}

// ScoringWeights holds the default weights for each score type.
type ScoringWeights struct {
	CodeQuality   float64 `yaml:"code_quality"`
	Performance   float64 `yaml:"performance"`
	Security      float64 `yaml:"security"`
	Documentation float64 `yaml:"documentation"`
	TestCoverage  float64 `yaml:"test_coverage"`
}

// Load reads configuration from the specified file path and applies environment overrides.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if val := os.Getenv("SCORING_DB_HOST"); val != "" {
		cfg.Database.Host = val
	}
	if val := os.Getenv("SCORING_DB_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			cfg.Database.Port = port
		}
	}
	if val := os.Getenv("SCORING_DB_USER"); val != "" {
		cfg.Database.User = val
	}
	if val := os.Getenv("SCORING_DB_PASSWORD"); val != "" {
		cfg.Database.Password = val
	}
	if val := os.Getenv("SCORING_DB_NAME"); val != "" {
		cfg.Database.DBName = val
	}
	if val := os.Getenv("SCORING_SERVER_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			cfg.Server.Port = port
		}
	}
	if val := os.Getenv("SCORING_GRPC_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			cfg.GRPC.Port = port
		}
	}
	if val := os.Getenv("SCORING_LOG_LEVEL"); val != "" {
		cfg.Logging.Level = val
	}
}

// DSN constructs the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// HTTPAddr returns the full HTTP server address.
func (c *ServerConfig) HTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GRPCAddr returns the full gRPC server address.
func (c *GRPCConfig) GRPCAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
