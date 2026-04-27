package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig   `mapstructure:"server"`
	Database  DatabaseConfig `mapstructure:"database"`
	RabbitMQ  RabbitMQConfig `mapstructure:"rabbitmq"`
	Email     EmailConfig    `mapstructure:"email"`
	Firebase  FirebaseConfig `mapstructure:"firebase"`
	SMS       SMSConfig      `mapstructure:"sms"`
	Templates TemplateConfig `mapstructure:"templates"`
	Log       LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RabbitMQConfig struct {
	URL           string `mapstructure:"url"`
	Exchange      string `mapstructure:"exchange"`
	Queue         string `mapstructure:"queue"`
	RoutingKey    string `mapstructure:"routing_key"`
	PrefetchCount int    `mapstructure:"prefetch_count"`
}

type EmailConfig struct {
	SMTPHost   string        `mapstructure:"smtp_host"`
	SMTPPort   int           `mapstructure:"smtp_port"`
	Username   string        `mapstructure:"username"`
	Password   string        `mapstructure:"password"`
	From       string        `mapstructure:"from"`
	FromName   string        `mapstructure:"from_name"`
	Path       string        `mapstructure:"path"`
	MaxRetries int           `mapstructure:"max_retries"`
	Timeout    time.Duration `mapstructure:"timeout"`
}

type FirebaseConfig struct {
	ProjectID         string `mapstructure:"project_id"`
	ServiceAccountKey string `mapstructure:"service_account_key"`
}

type SMSConfig struct {
	Provider   string `mapstructure:"provider"`
	AccountSID string `mapstructure:"account_sid"`
	AuthToken  string `mapstructure:"auth_token"`
	FromNumber string `mapstructure:"from_number"`
}

type TemplateConfig struct {
	Path string `mapstructure:"path"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load reads configuration from the given file path and environment variables.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	v.AutomaticEnv()

	// Bind environment variables with NOTIFICATION_ prefix
	v.SetEnvPrefix("NOTIFICATION")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
