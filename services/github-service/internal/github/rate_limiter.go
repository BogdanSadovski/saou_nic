package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements a token bucket rate limiter for GitHub API requests
type RateLimiter struct {
	limiter    *rate.Limiter
	mu         sync.RWMutex
	lastReset  time.Time
	resetAt    time.Time
	remaining  int
	totalLimit int
	burstSize  int
}

// NewRateLimiter creates a new rate limiter with the given requests per minute and burst size
func NewRateLimiter(requestsPerMinute, burstSize int) (*RateLimiter, error) {
	if requestsPerMinute <= 0 {
		return nil, fmt.Errorf("requests per minute must be positive")
	}
	if burstSize <= 0 {
		return nil, fmt.Errorf("burst size must be positive")
	}

	interval := time.Minute / time.Duration(requestsPerMinute)
	limiter := rate.NewLimiter(rate.Every(interval), burstSize)

	now := time.Now()
	return &RateLimiter{
		limiter:    limiter,
		lastReset:  now,
		resetAt:    now.Add(time.Minute),
		remaining:  requestsPerMinute,
		totalLimit: requestsPerMinute,
		burstSize:  burstSize,
	}, nil
}

// Wait blocks until the rate limiter allows an event to happen
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	r.remaining--
	return nil
}

// WaitN blocks until the rate limiter allows n events to happen
func (r *RateLimiter) WaitN(ctx context.Context, n int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.limiter.WaitN(ctx, n); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	r.remaining -= n
	return nil
}

// Allow returns true if an event can happen now
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	allowed := r.limiter.Allow()
	if allowed {
		r.remaining--
	}
	return allowed
}

// Remaining returns the approximate number of remaining requests
func (r *RateLimiter) Remaining() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.remaining
}

// ResetAt returns when the rate limit will reset
func (r *RateLimiter) ResetAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resetAt
}

// UpdateFromHeader updates the rate limiter state from GitHub API response headers
func (r *RateLimiter) UpdateFromHeader(remaining int, resetTime time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.remaining = remaining
	r.resetAt = resetTime
	r.lastReset = time.Now()
}

// Limit returns the configured requests per minute
func (r *RateLimiter) Limit() int {
	return r.totalLimit
}

// Burst returns the configured burst size
func (r *RateLimiter) Burst() int {
	return r.burstSize
}
