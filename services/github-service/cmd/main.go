package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/real-ass/github-service/internal/api"
	"github.com/real-ass/github-service/internal/config"
	ghclient "github.com/real-ass/github-service/internal/github"
	"github.com/real-ass/github-service/internal/repository"
	"github.com/real-ass/github-service/internal/service"
)

func main() {
	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	logger.Info("starting github-service",
		zap.String("version", "1.0.0"),
		zap.String("config_path", cfgPath),
	)

	// Initialize PostgreSQL repository
	repo, err := repository.NewPostgresRepository(&cfg.Postgres)
	if err != nil {
		logger.Fatal("failed to initialize postgres repository", zap.Error(err))
	}

	// Initialize GitHub client
	client, err := ghclient.NewClient(&cfg.GitHub, logger)
	if err != nil {
		logger.Fatal("failed to initialize github client", zap.Error(err))
	}

	// Initialize GitHub service
	githubService := service.NewGitHubService(client, repo, &cfg.GitHub, logger)

	// Setup router
	router := api.SetupRouter(githubService, &cfg.Server, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("server starting", zap.String("addr", cfg.Server.Addr()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited gracefully")
}

func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}
