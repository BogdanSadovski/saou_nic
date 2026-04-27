package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the analytics service.
type Config struct {
	Server     ServerConfig
	Postgres   PostgresConfig
	ClickHouse ClickHouseConfig
	Kafka      KafkaConfig
	Service    ServiceConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	MaxConns    int
	MinConns    int
	ConnTimeout time.Duration
}

// ClickHouseConfig holds ClickHouse connection configuration.
type ClickHouseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Timeout  time.Duration
}

// KafkaConfig holds Kafka consumer configuration.
type KafkaConfig struct {
	Brokers       []string
	ConsumerGroup string
	Topics        []string
	MinBytes      int
	MaxBytes      int
	MaxWait       time.Duration
}

// ServiceConfig holds business logic configuration.
type ServiceConfig struct {
	AggregationInterval   time.Duration
	RetentionDays         int
	ExportPageSizeMax     int
	DashboardCacheTTL     time.Duration
	RealtimeWindowMinutes int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvAsInt("SERVER_PORT", 8082),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Postgres: PostgresConfig{
			Host:        getEnv("POSTGRES_HOST", "localhost"),
			Port:        getEnvAsInt("POSTGRES_PORT", 5432),
			User:        getEnv("POSTGRES_USER", "analytics"),
			Password:    getEnv("POSTGRES_PASSWORD", "analytics_secret"),
			DBName:      getEnv("POSTGRES_DB", "analytics"),
			SSLMode:     getEnv("POSTGRES_SSLMODE", "disable"),
			MaxConns:    getEnvAsInt("POSTGRES_MAX_CONNS", 25),
			MinConns:    getEnvAsInt("POSTGRES_MIN_CONNS", 5),
			ConnTimeout: getEnvAsDuration("POSTGRES_CONN_TIMEOUT", 5*time.Second),
		},
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     getEnvAsInt("CLICKHOUSE_PORT", 9000),
			User:     getEnv("CLICKHOUSE_USER", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
			DBName:   getEnv("CLICKHOUSE_DB", "analytics"),
			Timeout:  getEnvAsDuration("CLICKHOUSE_TIMEOUT", 30*time.Second),
		},
		Kafka: KafkaConfig{
			Brokers:       getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "analytics-service"),
			Topics:        getEnvAsSlice("KAFKA_TOPICS", []string{"events", "user-activity"}),
			MinBytes:      getEnvAsInt("KAFKA_MIN_BYTES", 10e3),
			MaxBytes:      getEnvAsInt("KAFKA_MAX_BYTES", 10e6),
			MaxWait:       getEnvAsDuration("KAFKA_MAX_WAIT", 10*time.Second),
		},
		Service: ServiceConfig{
			AggregationInterval:   getEnvAsDuration("AGGREGATION_INTERVAL", 5*time.Minute),
			RetentionDays:         getEnvAsInt("RETENTION_DAYS", 90),
			ExportPageSizeMax:     getEnvAsInt("EXPORT_PAGE_SIZE_MAX", 10000),
			DashboardCacheTTL:     getEnvAsDuration("DASHBOARD_CACHE_TTL", 5*time.Minute),
			RealtimeWindowMinutes: getEnvAsInt("REALTIME_WINDOW_MINUTES", 15),
		},
	}
}

// DSN returns the PostgreSQL connection string.
func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, int(c.ConnTimeout.Seconds()),
	)
}

// DSN returns the ClickHouse connection string.
func (c *ClickHouseConfig) DSN() string {
	return fmt.Sprintf(
		"tcp://%s:%d?username=%s&password=%s&database=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

// Addr returns the HTTP server address.
func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Environment variable helpers.

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvAsSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		result := strings.Split(v, ",")
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}
