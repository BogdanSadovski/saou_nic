package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	GitHub   GitHubConfig   `mapstructure:"github"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Cache    CacheConfig    `mapstructure:"cache"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type GitHubConfig struct {
	BaseURL     string        `mapstructure:"base_url"`
	AccessToken string        `mapstructure:"access_token"`
	PerPage     int           `mapstructure:"per_page"`
	Timeout     time.Duration `mapstructure:"timeout"`
	RateLimit   RateLimitCfg  `mapstructure:"rate_limit"`
}

type RateLimitCfg struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
	BurstSize         int `mapstructure:"burst_size"`
}

type PostgresConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	DBName          string        `mapstructure:"dbname"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type CacheConfig struct {
	TTL             time.Duration `mapstructure:"ttl"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8082)
	v.SetDefault("server.mode", "release")
	v.SetDefault("github.per_page", 100)
	v.SetDefault("github.timeout", "30s")
	v.SetDefault("github.rate_limit.requests_per_minute", 500)
	v.SetDefault("github.rate_limit.burst_size", 50)
	v.SetDefault("postgres.ssl_mode", "disable")
	v.SetDefault("postgres.max_open_conns", 25)
	v.SetDefault("postgres.max_idle_conns", 5)
	v.SetDefault("postgres.conn_max_lifetime", "5m")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("cache.ttl", "10m")
	v.SetDefault("cache.cleanup_interval", "5m")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.GitHub.AccessToken = expandEnv(cfg.GitHub.AccessToken)
	cfg.Postgres.Host = expandEnv(cfg.Postgres.Host)
	cfg.Postgres.DBName = expandEnv(cfg.Postgres.DBName)
	cfg.Postgres.User = expandEnv(cfg.Postgres.User)
	cfg.Postgres.Password = expandEnv(cfg.Postgres.Password)

	return &cfg, nil
}

func expandEnv(val string) string {
	if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
		return os.Getenv(val[2 : len(val)-1])
	}
	return val
}

func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
