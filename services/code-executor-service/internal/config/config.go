package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Server   ServerConfig   `yaml:"server"`
	Executor ExecutorConfig `yaml:"executor"`
	Logging  LoggingConfig  `yaml:"logging"`
	Security SecurityConfig `yaml:"security"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Env     string `yaml:"env"`
}

type ServerConfig struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

type ExecutorConfig struct {
	MaxExecutionTime   time.Duration  `yaml:"max_execution_time"`
	MaxMemoryLimit     string         `yaml:"max_memory_limit"`
	MaxOutputSize      int64          `yaml:"max_output_size"`
	SandboxTimeout     time.Duration  `yaml:"sandbox_timeout"`
	CleanupInterval    time.Duration  `yaml:"cleanup_interval"`
	SupportedLanguages []string       `yaml:"supported_languages"`
	Python             LanguageConfig `yaml:"python"`
	JavaScript         LanguageConfig `yaml:"javascript"`
	Go                 LanguageConfig `yaml:"go"`
	Java               LanguageConfig `yaml:"java"`
}

type LanguageConfig struct {
	Binary      string        `yaml:"binary"`
	Timeout     time.Duration `yaml:"timeout"`
	MemoryLimit string        `yaml:"memory_limit"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type SecurityConfig struct {
	RequireToken       bool     `yaml:"require_token"`
	MaxCodeSize        int64    `yaml:"max_code_size"`
	DisallowedPatterns []string `yaml:"disallowed_patterns"`
}

func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := parseConfig(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseConfig(data []byte, cfg *Config) error {
	section := ""
	subsection := ""
	listTarget := ""

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := stripComment(scanner.Text())
		if strings.TrimSpace(line) == "" {
			continue
		}

		trimmed := strings.TrimSpace(line)
		indent := leadingSpaces(line)

		if strings.HasPrefix(trimmed, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			item = trimScalar(item)
			switch listTarget {
			case "supported_languages":
				cfg.Executor.SupportedLanguages = append(cfg.Executor.SupportedLanguages, item)
			case "disallowed_patterns":
				cfg.Security.DisallowedPatterns = append(cfg.Security.DisallowedPatterns, item)
			default:
				return fmt.Errorf("unexpected list item: %s", item)
			}
			continue
		}

		if strings.HasSuffix(trimmed, ":") {
			key := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			if indent == 0 {
				section = key
				subsection = ""
				listTarget = ""
				continue
			}
			if section == "executor" && indent == 2 && isExecutorLanguage(key) {
				subsection = key
				listTarget = ""
				continue
			}
			listTarget = key
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid config line: %s", trimmed)
		}

		key := strings.TrimSpace(parts[0])
		value := trimScalar(strings.TrimSpace(parts[1]))
		if value == "" {
			listTarget = key
			continue
		}

		switch section {
		case "app":
			if err := assignAppField(cfg, key, value); err != nil {
				return err
			}
		case "server":
			if err := assignServerField(cfg, key, value); err != nil {
				return err
			}
		case "executor":
			if subsection == "" {
				if key == "supported_languages" {
					listTarget = key
					continue
				}
				if err := assignExecutorField(cfg, key, value); err != nil {
					return err
				}
				continue
			}
			if err := assignLanguageField(getLanguageConfig(cfg, subsection), key, value); err != nil {
				return err
			}
		case "logging":
			if err := assignLoggingField(cfg, key, value); err != nil {
				return err
			}
		case "security":
			if err := assignSecurityField(cfg, key, value); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown config section: %s", section)
		}
	}

	return scanner.Err()
}

func stripComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
}

func leadingSpaces(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func trimScalar(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

func isExecutorLanguage(key string) bool {
	switch key {
	case "python", "javascript", "go", "java":
		return true
	default:
		return false
	}
}

func assignAppField(cfg *Config, key, value string) error {
	switch key {
	case "name":
		cfg.App.Name = value
	case "version":
		cfg.App.Version = value
	case "env":
		cfg.App.Env = value
	default:
		return fmt.Errorf("unknown app key: %s", key)
	}
	return nil
}

func assignServerField(cfg *Config, key, value string) error {
	switch key {
	case "host":
		cfg.Server.Host = value
	case "port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid server.port: %w", err)
		}
		cfg.Server.Port = port
	case "read_timeout":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid server.read_timeout: %w", err)
		}
		cfg.Server.ReadTimeout = duration
	case "write_timeout":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid server.write_timeout: %w", err)
		}
		cfg.Server.WriteTimeout = duration
	case "max_header_bytes":
		maxHeaderBytes, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid server.max_header_bytes: %w", err)
		}
		cfg.Server.MaxHeaderBytes = maxHeaderBytes
	default:
		return fmt.Errorf("unknown server key: %s", key)
	}
	return nil
}

func assignExecutorField(cfg *Config, key, value string) error {
	switch key {
	case "max_execution_time":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid executor.max_execution_time: %w", err)
		}
		cfg.Executor.MaxExecutionTime = duration
	case "max_memory_limit":
		cfg.Executor.MaxMemoryLimit = value
	case "max_output_size":
		maxOutputSize, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid executor.max_output_size: %w", err)
		}
		cfg.Executor.MaxOutputSize = maxOutputSize
	case "sandbox_timeout":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid executor.sandbox_timeout: %w", err)
		}
		cfg.Executor.SandboxTimeout = duration
	case "cleanup_interval":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid executor.cleanup_interval: %w", err)
		}
		cfg.Executor.CleanupInterval = duration
	default:
		return fmt.Errorf("unknown executor key: %s", key)
	}
	return nil
}

func assignLanguageField(target *LanguageConfig, key, value string) error {
	switch key {
	case "binary":
		target.Binary = value
	case "timeout":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid language timeout: %w", err)
		}
		target.Timeout = duration
	case "memory_limit":
		target.MemoryLimit = value
	default:
		return fmt.Errorf("unknown language key: %s", key)
	}
	return nil
}

func assignLoggingField(cfg *Config, key, value string) error {
	switch key {
	case "level":
		cfg.Logging.Level = value
	case "format":
		cfg.Logging.Format = value
	default:
		return fmt.Errorf("unknown logging key: %s", key)
	}
	return nil
}

func assignSecurityField(cfg *Config, key, value string) error {
	switch key {
	case "require_token":
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid security.require_token: %w", err)
		}
		cfg.Security.RequireToken = boolValue
	case "max_code_size":
		maxCodeSize, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid security.max_code_size: %w", err)
		}
		cfg.Security.MaxCodeSize = maxCodeSize
	default:
		return fmt.Errorf("unknown security key: %s", key)
	}
	return nil
}

func getLanguageConfig(cfg *Config, subsection string) *LanguageConfig {
	switch subsection {
	case "python":
		return &cfg.Executor.Python
	case "javascript":
		return &cfg.Executor.JavaScript
	case "go":
		return &cfg.Executor.Go
	case "java":
		return &cfg.Executor.Java
	default:
		return &LanguageConfig{}
	}
}
