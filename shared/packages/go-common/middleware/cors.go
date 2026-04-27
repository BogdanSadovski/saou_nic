package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSConfig holds configuration for the CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to access the resource.
	// Use "*" to allow all origins.
	AllowedOrigins []string
	// AllowedMethods is a list of methods that are allowed when accessing the resource.
	AllowedMethods []string
	// AllowedHeaders is a list of headers that are allowed when accessing the resource.
	AllowedHeaders []string
	// ExposedHeaders is a list of headers that are exposed to the browser.
	ExposedHeaders []string
	// AllowCredentials indicates whether the response can be shared when credentials mode is "include".
	AllowCredentials bool
	// MaxAge is the time (in seconds) that the results of a preflight request can be cached.
	MaxAge int
}

// DefaultCORSConfig returns a CORSConfig with sensible defaults.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-Total-Count"},
		AllowCredentials: false,
		MaxAge:           3600,
	}
}

// CORS creates a CORS middleware for Fiber.
func CORS(cfg CORSConfig) fiber.Handler {
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"*"}
	}
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	}

	corsConfig := cors.Config{
		AllowOrigins:     joinStrings(cfg.AllowedOrigins, ","),
		AllowMethods:     joinStrings(cfg.AllowedMethods, ","),
		AllowHeaders:     joinStrings(cfg.AllowedHeaders, ","),
		ExposeHeaders:    joinStrings(cfg.ExposedHeaders, ","),
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}

	handler := cors.New(corsConfig)
	return handler
}

// joinStrings joins a slice of strings with the given separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
