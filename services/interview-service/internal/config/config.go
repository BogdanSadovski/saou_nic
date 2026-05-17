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
	Redis    RedisConfig    `yaml:"redis"`
	Log      LogConfig      `yaml:"log"`
	Auth     AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type AuthConfig struct {
	SecretKey     string        `yaml:"secret_key"`
	TokenExpiry   time.Duration `yaml:"token_expiry"`
	RefreshExpiry time.Duration `yaml:"refresh_expiry"`
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

	// Override with environment variables if set
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	// DATABASE_URL takes precedence over individual vars (compose convention).
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		if u, err := url.Parse(raw); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
			if h := u.Hostname(); h != "" {
				cfg.Database.Host = h
			}
			if p := u.Port(); p != "" {
				if n, err := strconv.Atoi(p); err == nil {
					cfg.Database.Port = n
				}
			}
			if u.User != nil {
				if name := u.User.Username(); name != "" {
					cfg.Database.User = name
				}
				if pwd, ok := u.User.Password(); ok && pwd != "" {
					cfg.Database.Password = pwd
				}
			}
			if name := strings.TrimPrefix(u.Path, "/"); name != "" {
				// strip ?schema=... query
				if idx := strings.Index(name, "?"); idx >= 0 {
					name = name[:idx]
				}
				cfg.Database.DBName = name
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
		cfg.Database.Host = v
	}
	if v := pickEnv("POSTGRES_PORT", "DB_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = n
		}
	}
	if v := pickEnv("POSTGRES_USER", "DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := pickEnv("POSTGRES_PASSWORD", "DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := pickEnv("POSTGRES_DB", "DB_NAME"); v != "" {
		cfg.Database.DBName = v
	}

	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}

	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		if p, err := strconv.Atoi(redisPort); err == nil {
			cfg.Redis.Port = p
		}
	}

	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Auth.SecretKey = jwtSecret
	}

	return &cfg, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
