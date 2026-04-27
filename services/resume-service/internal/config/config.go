package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	S3       S3Config
	NLP      NLPConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
}

type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	UseSSL          bool
}

type NLPConfig struct {
	ModelPath     string
	MaxTokens     int
	EnableNER     bool
	EnableParsing bool
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// LoadConfig reads configuration from file or environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.dbname", "resume_db")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_conns", "10")

	v.SetDefault("s3.endpoint", "http://localhost:9000")
	v.SetDefault("s3.region", "us-east-1")
	v.SetDefault("s3.bucket", "resumes")
	v.SetDefault("s3.use_ssl", false)

	v.SetDefault("nlp.model_path", "./models")
	v.SetDefault("nlp.max_tokens", 512)
	v.SetDefault("nlp.enable_ner", true)
	v.SetDefault("nlp.enable_parsing", true)

	// Read config file
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		// Config file is optional; fallback to env vars and defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Bind environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("RESUME")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Defensive defaults for fields that may decode as zero-values in some env/file combinations.
	if cfg.Database.MaxConns <= 0 {
		cfg.Database.MaxConns = 10
	}
	if cfg.Database.Host == "" {
		cfg.Database.Host = "postgres"
	}
	if cfg.Database.Port == "" {
		cfg.Database.Port = "5432"
	}
	if cfg.Database.User == "" {
		cfg.Database.User = "postgres"
	}
	if cfg.Database.Password == "" {
		cfg.Database.Password = "postgres_secret"
	}
	if cfg.Database.DBName == "" {
		cfg.Database.DBName = "platform_db"
	}
	if cfg.S3.Endpoint == "" {
		cfg.S3.Endpoint = "http://minio:9000"
	}
	if cfg.S3.AccessKeyID == "" {
		cfg.S3.AccessKeyID = "minioadmin"
	}
	if cfg.S3.SecretAccessKey == "" {
		cfg.S3.SecretAccessKey = "minioadmin"
	}
	if cfg.S3.Region == "" {
		cfg.S3.Region = "us-east-1"
	}
	if cfg.S3.Bucket == "" {
		cfg.S3.Bucket = "resumes"
	}

	return &cfg, nil
}
