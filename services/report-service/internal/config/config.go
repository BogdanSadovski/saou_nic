package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
	"os"
)

// Config holds all configuration for the report-service
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	S3        S3Config        `yaml:"s3"`
	Generator GeneratorConfig `yaml:"generator"`
	Service   ServiceConfig   `yaml:"service"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string        `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"ssl_mode"`
	MaxConns int    `yaml:"max_conns"`
}

// DSN returns the PostgreSQL connection string
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// S3Config holds S3-compatible storage configuration
type S3Config struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	UseSSL          bool   `yaml:"use_ssl"`
}

// GeneratorConfig holds report generator configuration
type GeneratorConfig struct {
	TemplateDir    string `yaml:"template_dir"`
	AssetsDir      string `yaml:"assets_dir"`
	DefaultFormat  string `yaml:"default_format"`
	MaxFileSizeMB  int    `yaml:"max_file_size_mb"`
}

// ServiceConfig holds report service configuration
type ServiceConfig struct {
	ReportRetentionDays int `yaml:"report_retention_days"`
	MaxConcurrentJobs   int `yaml:"max_concurrent_jobs"`
}

// LoadConfig reads and parses the configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply defaults
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8082"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 15 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 60 * time.Second
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.MaxConns == 0 {
		cfg.Database.MaxConns = 25
	}
	if cfg.Generator.TemplateDir == "" {
		cfg.Generator.TemplateDir = "./internal/templates"
	}
	if cfg.Generator.AssetsDir == "" {
		cfg.Generator.AssetsDir = "./assets"
	}
	if cfg.Generator.DefaultFormat == "" {
		cfg.Generator.DefaultFormat = "pdf"
	}
	if cfg.Generator.MaxFileSizeMB == 0 {
		cfg.Generator.MaxFileSizeMB = 50
	}
	if cfg.Service.ReportRetentionDays == 0 {
		cfg.Service.ReportRetentionDays = 90
	}
	if cfg.Service.MaxConcurrentJobs == 0 {
		cfg.Service.MaxConcurrentJobs = 10
	}

	// Override with environment variables if set
	if env := os.Getenv("REPORT_SERVICE_PORT"); env != "" {
		cfg.Server.Port = env
	}
	if env := os.Getenv("REPORT_DB_HOST"); env != "" {
		cfg.Database.Host = env
	}
	if env := os.Getenv("REPORT_DB_USER"); env != "" {
		cfg.Database.User = env
	}
	if env := os.Getenv("REPORT_DB_PASSWORD"); env != "" {
		cfg.Database.Password = env
	}
	if env := os.Getenv("REPORT_DB_NAME"); env != "" {
		cfg.Database.DBName = env
	}
	if env := os.Getenv("REPORT_S3_ENDPOINT"); env != "" {
		cfg.S3.Endpoint = env
	}
	if env := os.Getenv("REPORT_S3_BUCKET"); env != "" {
		cfg.S3.Bucket = env
	}

	return &cfg, nil
}
