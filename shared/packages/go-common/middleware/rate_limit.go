package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// newTokenBucket creates a new token bucket with the given max tokens and refill rate.
func newTokenBucket(maxTokens float64, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit.
func (tb *tokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// RateLimiterConfig holds configuration for the rate limiter middleware.
type RateLimiterConfig struct {
	// MaxTokens is the maximum number of tokens (burst capacity).
	MaxTokens float64
	// RefillRate is the number of tokens added per second.
	RefillRate float64
	// KeyFunc extracts the key for rate limiting (e.g., IP address, user ID).
	// If nil, defaults to the client IP.
	KeyFunc func(c *fiber.Ctx) string
	// ErrorHandler is called when the rate limit is exceeded.
	ErrorHandler func(c *fiber.Ctx) error
}

// DefaultRateLimiterConfig returns a RateLimiterConfig with sensible defaults.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		MaxTokens:  100,
		RefillRate: 10,
		KeyFunc:    nil,
		ErrorHandler: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "rate limit exceeded",
			})
		},
	}
}

// RateLimiter creates a token bucket rate limiter middleware for Fiber.
func RateLimiter(cfg RateLimiterConfig) fiber.Handler {
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 100
	}
	if cfg.RefillRate <= 0 {
		cfg.RefillRate = 10
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *fiber.Ctx) string {
			return c.IP()
		}
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = DefaultRateLimiterConfig().ErrorHandler
	}

	buckets := make(map[string]*tokenBucket)
	var mu sync.Mutex

	getBucket := func(key string) *tokenBucket {
		mu.Lock()
		defer mu.Unlock()

		bucket, exists := buckets[key]
		if !exists {
			bucket = newTokenBucket(cfg.MaxTokens, cfg.RefillRate)
			buckets[key] = bucket
		}
		return bucket
	}

	return func(c *fiber.Ctx) error {
		key := cfg.KeyFunc(c)
		bucket := getBucket(key)

		if !bucket.Allow() {
			return cfg.ErrorHandler(c)
		}

		return c.Next()
	}
}
