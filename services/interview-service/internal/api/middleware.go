package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"
	contextKeyTier   contextKey = "subscription_tier"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	// SubscriptionTier — задаётся user-service'ом при выдаче JWT, либо
	// синхронизируется через api-gateway. Используется quota-helper'ом
	// для определения лимитов. Пустое значение трактуется как trial.
	SubscriptionTier string `json:"tier,omitempty"`
	jwt.RegisteredClaims
}

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Logging middleware logs request details
func Logging(logger *logrus.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			logger.WithFields(logrus.Fields{
				"method": r.Method,
				"path":   r.URL.Path,
				"remote": r.RemoteAddr,
			}).Info("request started")

			next.ServeHTTP(w, r)

			logger.WithFields(logrus.Fields{
				"method":   r.Method,
				"path":     r.URL.Path,
				"duration": time.Since(start),
			}).Info("request completed")
		})
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS() Middleware {
	allowedOrigins := strings.Split(strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")), ",")
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			originSet[o] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := originSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Recovery middleware recovers from panics
func Recovery(logger *logrus.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithField("error", err).Error("panic recovered")
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates JWT tokens
type AuthMiddleware struct {
	secretKey string
	logger    *logrus.Logger
}

func NewAuthMiddleware(secretKey string, logger *logrus.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		secretKey: secretKey,
		logger:    logger,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenString := ""
		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				writeError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}
		} else {
			tokenString = strings.TrimSpace(r.URL.Query().Get("access_token"))
			if tokenString == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization token")
				return
			}
		}

		claims, err := m.validateToken(tokenString)
		if err != nil {
			m.logger.WithError(err).Warn("invalid token")
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
		// Тариф: JWT claim приоритетнее. Если его нет — пробуем
		// заголовок X-Subscription-Tier (api-gateway может его
		// проставлять, обогащая запрос данными из admin-service).
		tier := claims.SubscriptionTier
		if strings.TrimSpace(tier) == "" {
			tier = strings.TrimSpace(r.Header.Get("X-Subscription-Tier"))
		}
		ctx = context.WithValue(ctx, contextKeyTier, tier)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

// RequireRole middleware checks if the user has the required role
func RequireRole(role string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value(ContextKeyRole).(string)
			if !ok || userRole != role {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit middleware (placeholder implementation)
func RateLimit(maxRequests int, window time.Duration) Middleware {
	// In production, use Redis or a proper rate limiter
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Placeholder: implement actual rate limiting
			next.ServeHTTP(w, r)
		})
	}
}
