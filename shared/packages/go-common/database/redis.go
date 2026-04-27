package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
)

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Address      string        `yaml:"address" json:"address"`
	Password     string        `yaml:"password" json:"password"`
	DB           int           `yaml:"db" json:"db"`
	PoolSize     int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns" json:"min_idle_conns"`
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	PoolTimeout  time.Duration `yaml:"pool_timeout" json:"pool_timeout"`
}

// DefaultRedisConfig returns a RedisConfig with sensible defaults.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Address:      "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
}

// Redis represents a Redis client connection.
type Redis struct {
	client *redis.Client
	config RedisConfig
}

// NewRedis creates a new Redis instance without connecting.
func NewRedis(cfg RedisConfig) *Redis {
	return &Redis{config: cfg}
}

// Connect establishes a connection to Redis.
func (r *Redis) Connect(ctx context.Context) error {
	r.client = redis.NewClient(&redis.Options{
		Addr:         r.config.Address,
		Password:     r.config.Password,
		DB:           r.config.DB,
		PoolSize:     r.config.PoolSize,
		MinIdleConns: r.config.MinIdleConns,
		DialTimeout:  r.config.DialTimeout,
		ReadTimeout:  r.config.ReadTimeout,
		WriteTimeout: r.config.WriteTimeout,
		PoolTimeout:  r.config.PoolTimeout,
	})

	// Verify connection
	if err := r.client.Ping(ctx).Err(); err != nil {
		r.client.Close()
		r.client = nil
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("connected to Redis",
		zap.String("address", r.config.Address),
		zap.Int("db", r.config.DB),
	)

	return nil
}

// Close closes the Redis connection.
func (r *Redis) Close() error {
	if r.client != nil {
		err := r.client.Close()
		if err != nil {
			return fmt.Errorf("failed to close Redis connection: %w", err)
		}
		logger.Info("Redis connection closed")
		r.client = nil
	}
	return nil
}

// Ping checks if the Redis connection is alive.
func (r *Redis) Ping(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("Redis not connected")
	}
	return r.client.Ping(ctx).Err()
}

// Client returns the underlying Redis client.
func (r *Redis) Client() *redis.Client {
	return r.client
}

// Get retrieves a value from Redis.
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set stores a value in Redis with an optional expiration.
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Delete removes a key from Redis.
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis.
func (r *Redis) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// SetWithNX sets a key only if it does not exist.
func (r *Redis) SetWithNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// Increment increments the integer value of a key.
func (r *Redis) Increment(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire sets a timeout on a key.
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return r.client.Expire(ctx, key, expiration).Result()
}

// Keys returns all keys matching a pattern.
func (r *Redis) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}
