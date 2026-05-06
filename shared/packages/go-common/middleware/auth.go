package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// ContextKeyUserID is the context key for the authenticated user ID.
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyUserEmail is the context key for the authenticated user email.
	ContextKeyUserEmail ContextKey = "user_email"
	// ContextKeyUserClaims is the context key for all JWT claims.
	ContextKeyUserClaims ContextKey = "user_claims"
)

// TokenValidator is a function that validates a JWT token.
type TokenValidator func(token *jwt.Token) (*jwt.RegisteredClaims, error)

// JWTAuthConfig holds configuration for the JWT auth middleware.
type JWTAuthConfig struct {
	// Validator is an optional function for additional claims validation.
	// Signature validation is performed by the middleware using SigningKey.
	Validator TokenValidator
	// SigningKey is the HMAC secret used to verify token signatures.
	// REQUIRED.
	SigningKey []byte
	// TokenLookup is the header/query/cookie to look for the token.
	// Format: "header:<name>" or "query:<name>" or "cookie:<name>"
	// Default: "header:Authorization"
	TokenLookup string
	// AuthScheme is the scheme to use (e.g., "Bearer").
	// Default: "Bearer"
	AuthScheme string
	// SkipPaths is a list of paths to skip authentication.
	SkipPaths []string
}

// DefaultJWTAuthConfig returns a JWTAuthConfig with sensible defaults.
func DefaultJWTAuthConfig(validator TokenValidator) JWTAuthConfig {
	return JWTAuthConfig{
		Validator:   validator,
		TokenLookup: "header:Authorization",
		AuthScheme:  "Bearer",
		SkipPaths:   []string{},
	}
}

// JWTAuth creates a JWT authentication middleware for Fiber.
func JWTAuth(cfg JWTAuthConfig) fiber.Handler {
	if len(cfg.SigningKey) == 0 {
		panic("middleware.JWTAuth: SigningKey is required")
	}
	if cfg.TokenLookup == "" {
		cfg.TokenLookup = "header:Authorization"
	}
	if cfg.AuthScheme == "" {
		cfg.AuthScheme = "Bearer"
	}

	skipPaths := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()
		if skipPaths[path] {
			return c.Next()
		}

		var tokenStr string
		parts := strings.SplitN(cfg.TokenLookup, ":", 2)
		if len(parts) != 2 {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token lookup configuration",
			})
		}

		switch parts[0] {
		case "header":
			tokenStr = extractFromHeader(c, parts[1], cfg.AuthScheme)
		case "query":
			tokenStr = c.Query(parts[1])
		case "cookie":
			tokenStr = c.Cookies(parts[1])
		default:
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token lookup source",
			})
		}

		if tokenStr == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authentication token",
			})
		}

		token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return cfg.SigningKey, nil
		})
		if err != nil || !token.Valid {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authentication token",
			})
		}

		claims, ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token claims",
			})
		}

		if cfg.Validator != nil {
			validated, err := cfg.Validator(token)
			if err != nil {
				return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
					"error": "invalid authentication token: " + err.Error(),
				})
			}
			if validated != nil {
				claims = validated
			}
		}

		ctx := context.WithValue(c.Context(), ContextKeyUserID, claims.Subject)
		c.SetUserContext(ctx)
		c.Locals(string(ContextKeyUserID), claims.Subject)
		c.Locals(string(ContextKeyUserEmail), claims.Subject)
		c.Locals(string(ContextKeyUserClaims), claims)

		return c.Next()
	}
}


// extractFromHeader extracts the token from an Authorization header.
func extractFromHeader(c *fiber.Ctx, headerName, authScheme string) string {
	authHeader := c.Get(headerName)
	if authHeader == "" {
		return ""
	}
	if authScheme == "" {
		return authHeader
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], authScheme) {
		return ""
	}
	return parts[1]
}

// GetUserIDFromContext retrieves the user ID from the context.
func GetUserIDFromContext(c *fiber.Ctx) string {
	if id, ok := c.Locals(string(ContextKeyUserID)).(string); ok {
		return id
	}
	return ""
}

// GetUserEmailFromContext retrieves the user email from the context.
func GetUserEmailFromContext(c *fiber.Ctx) string {
	if email, ok := c.Locals(string(ContextKeyUserEmail)).(string); ok {
		return email
	}
	return ""
}
