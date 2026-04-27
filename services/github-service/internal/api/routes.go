package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/real-ass/github-service/internal/config"
	"github.com/real-ass/github-service/internal/service"
)

// SetupRouter configures the Gin router with all routes
func SetupRouter(
	githubService *service.GitHubService,
	cfg *config.ServerConfig,
	logger *zap.Logger,
) *gin.Engine {
	// Set Gin mode
	gin.SetMode(cfg.Mode)

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(RequestLogger(logger))
	router.Use(CORSMiddleware())

	// Health and readiness endpoints
	router.GET("/health", NewHandlers(githubService, logger).HealthCheck)
	router.GET("/ready", NewHandlers(githubService, logger).ReadyCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		h := NewHandlers(githubService, logger)

		// Repository endpoints
		repos := v1.Group("/repositories")
		{
			repos.GET("", h.ListRepositories)
			repos.GET("/:id", h.GetRepository)
			repos.POST("/sync/:owner/:name", h.SyncRepository)

			// Repository-specific endpoints
			repo := repos.Group("/:id")
			{
				// Contributors
				repo.POST("/contributors/sync", h.SyncContributors)
				repo.GET("/contributors/top", h.GetTopContributors)

				// Pull Requests
				repo.POST("/pull-requests/sync", h.SyncPullRequests)

				// Analysis
				repo.POST("/analyze", h.AnalyzeRepository)
				repo.POST("/contributions/analyze", h.AnalyzeContributions)
			}
		}
	}

	return router
}

// RequestLogger middleware for structured request logging
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start).String()

		logger.Info("request completed",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.String("latency", latency),
		)
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
