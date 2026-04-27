package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/real-ass/user-service/internal/api"
	"github.com/real-ass/user-service/internal/config"
	grpcs "github.com/real-ass/user-service/internal/grpc"
	"github.com/real-ass/user-service/internal/repository"
	"github.com/real-ass/user-service/internal/service"
	jwtpkg "github.com/real-ass/user-service/pkg/jwt"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database repository
	userRepo, err := repository.NewPostgresRepository(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if closer, ok := userRepo.(interface{ Close() }); ok {
			closer.Close()
		}
	}()

	// Initialize services
	tokenManager := jwtpkg.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
	)

	authService := service.NewAuthService(userRepo, tokenManager)
	userService := service.NewUserService(userRepo)
	oauthService := service.NewOAuthService(userRepo, tokenManager, service.OAuthConfig{
		GoogleClientID:     cfg.OAuth.GoogleClientID,
		GoogleClientSecret: cfg.OAuth.GoogleClientSecret,
		GoogleRedirectURL:  cfg.OAuth.GoogleRedirectURL,
		GitHubClientID:     cfg.OAuth.GitHubClientID,
		GitHubClientSecret: cfg.OAuth.GitHubClientSecret,
		GitHubRedirectURL:  cfg.OAuth.GitHubRedirectURL,
	})

	// Initialize handlers
	handler := api.NewHandler(authService, userService, oauthService)

	// Setup router
	router := handler.RegisterRoutes()

	// Apply auth middleware
	authMiddleware := api.NewAuthMiddleware(authService)
	_ = authMiddleware
	// Apply middleware to protected routes as needed

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Create gRPC server
	userHandler := grpcs.NewUserHandler(userService)
	grpcServer := grpcs.NewGRPCServer(userHandler, cfg.Server.GRPCPort)
	grpcServer.RegisterServices()

	// Start gRPC server in a goroutine
	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("HTTP server starting on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	grpcServer.Stop()

	log.Println("Servers stopped")
}
