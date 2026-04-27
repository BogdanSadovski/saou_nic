package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
)

// RecoveryConfig holds configuration for the recovery middleware.
type RecoveryConfig struct {
	// EnableStackTrace enables/disables stack trace capture.
	EnableStackTrace bool
	// ErrorHandler is called when a panic is recovered.
	// If nil, a default 500 response is returned.
	ErrorHandler func(c *fiber.Ctx, err interface{}) error
}

// DefaultRecoveryConfig returns a RecoveryConfig with sensible defaults.
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		EnableStackTrace: true,
		ErrorHandler:     nil,
	}
}

// Recovery creates a panic recovery middleware for Fiber.
func Recovery() fiber.Handler {
	return RecoveryWithConfig(DefaultRecoveryConfig())
}

// RecoveryWithConfig creates a panic recovery middleware with custom configuration.
func RecoveryWithConfig(cfg RecoveryConfig) fiber.Handler {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultRecoveryErrorHandler
	}

	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := ""
				if cfg.EnableStackTrace {
					stack = string(debug.Stack())
				}

				logger.Error("panic recovered",
					zap.Any("error", r),
					zap.String("path", c.Path()),
					zap.String("method", c.Method()),
					zap.String("stack", stack),
				)

				err = cfg.ErrorHandler(c, r)
			}
		}()

		return c.Next()
	}
}

// defaultRecoveryErrorHandler handles recovered panics with a generic 500 response.
func defaultRecoveryErrorHandler(c *fiber.Ctx, err interface{}) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "internal server error",
	})
}

// SafeExecute wraps a function that might panic and returns an error if it does.
func SafeExecute(fn func() error, description string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error("panic in safe execution",
				zap.Any("error", r),
				zap.String("description", description),
				zap.String("stack", stack),
			)
			err = fmt.Errorf("panic in %s: %v", description, r)
		}
	}()
	return fn()
}
