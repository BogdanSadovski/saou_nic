package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	OAuth    OAuthConfig    `yaml:"oauth"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	GRPCPort     int           `yaml:"grpc_port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Name            string        `yaml:"name"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type JWTConfig struct {
	Secret          string        `yaml:"secret"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`
}

type OAuthConfig struct {
	GoogleClientID     string `yaml:"google_client_id"`
	GoogleClientSecret string `yaml:"google_client_secret"`
	GoogleRedirectURL  string `yaml:"google_redirect_url"`
	GitHubClientID     string `yaml:"github_client_id"`
	GitHubClientSecret string `yaml:"github_client_secret"`
	GitHubRedirectURL  string `yaml:"github_redirect_url"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Environment variables override YAML so deployments don't need to
	// bake secrets/IPs into the image. Both POSTGRES_* (compose convention)
	// and DB_* (legacy) prefixes are honoured. DATABASE_URL takes
	// precedence over individual vars when set.
	applyDatabaseEnv(&cfg.Database)

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWT.Secret = jwtSecret
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func applyDatabaseEnv(db *DatabaseConfig) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		if u, err := url.Parse(raw); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
			if h := u.Hostname(); h != "" {
				db.Host = h
			}
			if p := u.Port(); p != "" {
				if n, err := strconv.Atoi(p); err == nil {
					db.Port = n
				}
			}
			if u.User != nil {
				if name := u.User.Username(); name != "" {
					db.User = name
				}
				if pwd, ok := u.User.Password(); ok && pwd != "" {
					db.Password = pwd
				}
			}
			if name := strings.TrimPrefix(u.Path, "/"); name != "" {
				db.Name = name
			}
		}
	}

	pickEnv := func(keys ...string) string {
		for _, k := range keys {
			if v := os.Getenv(k); v != "" {
				return v
			}
		}
		return ""
	}

	if v := pickEnv("POSTGRES_HOST", "DB_HOST"); v != "" {
		db.Host = v
	}
	if v := pickEnv("POSTGRES_PORT", "DB_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db.Port = n
		}
	}
	if v := pickEnv("POSTGRES_USER", "DB_USER"); v != "" {
		db.User = v
	}
	if v := pickEnv("POSTGRES_PASSWORD", "DB_PASSWORD"); v != "" {
		db.Password = v
	}
	if v := pickEnv("POSTGRES_DB", "DB_NAME"); v != "" {
		db.Name = v
	}
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	return nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}
