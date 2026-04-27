package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config provides a unified configuration loader that merges
// values from YAML files and environment variables.
type Config struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// New creates a new empty Config.
func New() *Config {
	return &Config{
		data: make(map[string]interface{}),
	}
}

// Load reads configuration from a YAML file.
// Environment variables take precedence over file values.
func Load(path string) (*Config, error) {
	cfg := New()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
		}

		var fileData map[string]interface{}
		if err := yaml.Unmarshal(data, &fileData); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
		}

		cfg.data = flattenMap(fileData, "")
	}

	// Override with environment variables
	cfg.loadEnvVars()

	return cfg, nil
}

// LoadYAML loads configuration from YAML bytes.
func LoadYAML(data []byte) (*Config, error) {
	cfg := New()

	var fileData map[string]interface{}
	if err := yaml.Unmarshal(data, &fileData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	cfg.data = flattenMap(fileData, "")
	cfg.loadEnvVars()

	return cfg, nil
}

// Get retrieves a configuration value by key.
// Keys use dot notation for nested values (e.g., "database.host").
func (c *Config) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[key]
}

// GetString retrieves a configuration value as a string.
func (c *Config) GetString(key string) string {
	val := c.Get(key)
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetStringOrDefault retrieves a string value or returns the default.
func (c *Config) GetStringOrDefault(key, defaultVal string) string {
	val := c.GetString(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// GetInt retrieves a configuration value as an integer.
func (c *Config) GetInt(key string) int {
	val := c.Get(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// GetIntOrDefault retrieves an int value or returns the default.
func (c *Config) GetIntOrDefault(key string, defaultVal int) int {
	val := c.GetInt(key)
	if val == 0 {
		return defaultVal
	}
	return val
}

// GetBool retrieves a configuration value as a boolean.
func (c *Config) GetBool(key string) bool {
	val := c.Get(key)
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(v)
		return lower == "true" || lower == "1" || lower == "yes"
	case int:
		return v != 0
	case float64:
		return v != 0
	}
	return false
}

// GetBoolOrDefault retrieves a bool value or returns the default.
func (c *Config) GetBoolOrDefault(key string, defaultVal bool) bool {
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return c.GetBool(key)
}

// GetFloat retrieves a configuration value as a float64.
func (c *Config) GetFloat(key string) float64 {
	val := c.Get(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

// GetStringSlice retrieves a configuration value as a string slice.
// Supports comma-separated strings or YAML arrays.
func (c *Config) GetStringSlice(key string) []string {
	val := c.Get(key)
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case string:
		if v == "" {
			return nil
		}
		return strings.Split(v, ",")
	}
	return nil
}

// Set sets a configuration value.
func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Has checks if a configuration key exists.
func (c *Config) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.data[key]
	return ok
}

// Keys returns all configuration keys.
func (c *Config) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// All returns the entire configuration map.
func (c *Config) All() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// flattenMap flattens a nested map into dot-notation keys.
func flattenMap(m map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]interface{}:
			for k, val := range flattenMap(v, fullKey) {
				result[k] = val
			}
		default:
			result[fullKey] = value
		}
	}
	return result
}

// loadEnvVars loads configuration from environment variables.
// Env vars are mapped to config keys using REAL_ASS_ prefix and underscore-to-dot conversion.
func (c *Config) loadEnvVars() {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := "REAL_ASS_"
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], prefix)
		key = strings.ToLower(strings.ReplaceAll(key, "_", "."))
		c.data[key] = parts[1]
	}
}
