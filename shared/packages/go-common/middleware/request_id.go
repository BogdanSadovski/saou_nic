package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	// HeaderRequestID is the header name for the request ID.
	HeaderRequestID = "X-Request-ID"
	// ContextKeyRequestID is the context key for the request ID.
	ContextKeyRequestID = "request_id"
)

// RequestIDConfig holds configuration for the request ID middleware.
type RequestIDConfig struct {
	// Header is the header name to use for the request ID.
	Header string
	// Generator is the function to generate a new request ID.
	// If nil, defaults to generating a UUID.
	Generator func() string
	// ContextKey is the context key to store the request ID.
	ContextKey string
}

// DefaultRequestIDConfig returns a RequestIDConfig with sensible defaults.
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		Header:     HeaderRequestID,
		Generator:  nil,
		ContextKey: ContextKeyRequestID,
	}
}

// RequestID creates a middleware that generates or propagates a request ID.
// If the request already contains an X-Request-ID header, it is used.
// Otherwise, a new UUID is generated.
func RequestID() fiber.Handler {
	return RequestIDWithConfig(DefaultRequestIDConfig())
}

// RequestIDWithConfig creates a request ID middleware with custom configuration.
func RequestIDWithConfig(cfg RequestIDConfig) fiber.Handler {
	if cfg.Header == "" {
		cfg.Header = HeaderRequestID
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = ContextKeyRequestID
	}
	if cfg.Generator == nil {
		cfg.Generator = func() string {
			return uuid.New().String()
		}
	}

	return func(c *fiber.Ctx) error {
		requestID := c.Get(cfg.Header)
		if requestID == "" {
			requestID = cfg.Generator()
		}

		// Set the request ID in the response header
		c.Set(cfg.Header, requestID)

		// Store in context for access in handlers
		c.Locals(cfg.ContextKey, requestID)

		return c.Next()
	}
}

// GetRequestID retrieves the request ID from the Fiber context.
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(ContextKeyRequestID).(string); ok {
		return id
	}
	return c.Get(HeaderRequestID)
}
