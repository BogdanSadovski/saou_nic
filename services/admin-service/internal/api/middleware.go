package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
	"github.com/real-ass/admin-service/pkg/rbac"
)

// Context keys
const (
	ContextKeyUserID    = "user_id"
	ContextKeyUserRole  = "user_role"
	ContextKeyUserEmail = "user_email"
)

// AuthMiddleware handles JWT authentication.
type AuthMiddleware struct {
	jwtSecret string
}

// NewAuthMiddleware creates a new AuthMiddleware.
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// Authenticate validates JWT token and sets user context.
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := m.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Set user context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserRole, claims.Role)
		c.Set(ContextKeyUserEmail, claims.Email)

		c.Next()
	}
}

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	UserID uuid.UUID
	Email  string
	Role   domain.UserRole
}

// validateToken validates and parses the JWT token.
func (m *AuthMiddleware) validateToken(tokenString string) (*TokenClaims, error) {
	// In production, use jwt.Parse with proper validation
	// This is a simplified stub
	if len(tokenString) < 10 {
		return nil, fmt.Errorf("invalid token")
	}

	// Stub: In production, parse JWT and verify signature
	// For now, return a mock claim
	return &TokenClaims{
		UserID: uuid.New(),
		Email:  "admin@example.com",
		Role:   domain.RoleSuperAdmin,
	}, nil
}

// RBACMiddleware handles role-based access control.
type RBACMiddleware struct {
	enabled bool
}

// NewRBACMiddleware creates a new RBACMiddleware.
func NewRBACMiddleware(enabled bool) *RBACMiddleware {
	return &RBACMiddleware{enabled: enabled}
}

// RequirePermission checks if the user has the required permission.
func (m *RBACMiddleware) RequirePermission(resource rbac.Resource, action rbac.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.enabled {
			c.Next()
			return
		}

		role, exists := c.Get(ContextKeyUserRole)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "user role not found in context"})
			c.Abort()
			return
		}

		userRole, ok := role.(domain.UserRole)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid user role in context"})
			c.Abort()
			return
		}

		if err := rbac.Enforce(userRole, resource, action); err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":     "permission denied",
				"required":  fmt.Sprintf("%s:%s", resource, action),
				"user_role": userRole,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole checks if the user has at least the specified role level.
func (m *RBACMiddleware) RequireRole(minRole domain.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.enabled {
			c.Next()
			return
		}

		role, exists := c.Get(ContextKeyUserRole)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "user role not found in context"})
			c.Abort()
			return
		}

		userRole, ok := role.(domain.UserRole)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid user role in context"})
			c.Abort()
			return
		}

		if !rbac.CanManageRole(userRole, minRole) && userRole != minRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error":     "insufficient role privileges",
				"required":  minRole,
				"user_role": userRole,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "false")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggerMiddleware logs HTTP requests.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// In production, use structured logging (e.g., zap, logrus)
		fmt.Printf("[HTTP] %d %s %s %s %s\n",
			status,
			c.Request.Method,
			path,
			query,
			latency,
		)
	}
}

// RecoveryMiddleware handles panics and returns 500.
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultWriter, func(c *gin.Context, err any) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	})
}

// RequestIDMiddleware adds a unique request ID to each request.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// GetUserIDFromContext retrieves the user ID from the gin context.
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user ID type in context")
	}

	return id, nil
}

// GetUserRoleFromContext retrieves the user role from the gin context.
func GetUserRoleFromContext(c *gin.Context) (domain.UserRole, error) {
	role, exists := c.Get(ContextKeyUserRole)
	if !exists {
		return "", fmt.Errorf("user role not found in context")
	}

	r, ok := role.(domain.UserRole)
	if !ok {
		return "", fmt.Errorf("invalid user role type in context")
	}

	return r, nil
}

// Add this import
var _ context.Context = context.Background()
