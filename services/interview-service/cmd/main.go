package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/interview-platform/interview-service/internal/api"
	"github.com/interview-platform/interview-service/internal/config"
	"github.com/interview-platform/interview-service/internal/domain"
	"github.com/interview-platform/interview-service/internal/repository"
	"github.com/interview-platform/interview-service/internal/service"
	"github.com/interview-platform/interview-service/internal/websocket"
	"github.com/interview-platform/interview-service/pkg/codeexecutor"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("starting interview-service")

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.WithError(err).Fatal("failed to load configuration")
	}

	// Initialize database
	repo, err := repository.NewPostgresRepository(&cfg.Database, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to initialize database")
	}
	defer repo.Close()

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Verify Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithError(err).Fatal("failed to connect to redis")
	}
	cancel()
	defer redisClient.Close()

	logger.Info("successfully connected to redis")

	// Initialize repository interfaces
	repositories := &domain.Repository{
		Interview: repository.NewInterviewRepoAdapter(repo),
		Question:  repository.NewQuestionRepoAdapter(repo),
		Session:   repository.NewSessionRepoAdapter(repo),
		Answer:    repository.NewAnswerRepoAdapter(repo),
	}

	// Initialize services
	questionGen := service.NewQuestionGenerator()
	interviewService := service.NewInterviewService(repositories, questionGen)
	sessionManager := service.NewSessionManager(repositories)

	codeExecutorURL := os.Getenv("CODE_EXECUTOR_URL")
	if codeExecutorURL == "" {
		codeExecutorURL = "http://code-executor-service:8083"
	}
	codeExecutorClient := codeexecutor.New(codeExecutorURL)

	// Initialize handlers
	apiHandler := api.NewHandler(interviewService, sessionManager, repo, repo, codeExecutorClient, redisClient, logger)
	wsHandler := websocket.NewHandler(logger)
	authMiddleware := api.NewAuthMiddleware(cfg.Auth.SecretKey, logger)

	// Setup router
	router := api.NewRouter(apiHandler, wsHandler, authMiddleware, logger)
	router.ApplyMiddleware()

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("addr", cfg.Server.Addr()).Info("starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("server forced to shutdown")
	}

	logger.Info("server stopped")
}
