package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/real-ass/admin-service/internal/api"
	"github.com/real-ass/admin-service/internal/config"
	grpcserver "github.com/real-ass/admin-service/internal/grpc"
	"github.com/real-ass/admin-service/internal/repository"
	"github.com/real-ass/admin-service/internal/service"
)

func main() {
	// Determine config path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database repository
	db, err := repository.NewPostgresRepository(ctx, &cfg.Database)
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Database connection established")

	userRepo := repository.NewUserRepositoryAdapter(db)
	subRepo := repository.NewSubscriptionRepositoryAdapter(db)
	auditRepo := repository.NewAuditLogRepositoryAdapter(db)
	roleRepo := repository.NewRoleRepositoryAdapter(db)

	// Initialize services
	userService := service.NewUserService(userRepo, auditRepo)
	subService := service.NewSubscriptionService(subRepo, userRepo, auditRepo)
	auditService := service.NewAuditService(auditRepo, userRepo)
	adminService := service.NewAdminService(userRepo, subRepo, auditRepo, roleRepo)

	// Initialize HTTP handlers
	handler := api.NewHandler(adminService, userService, subService, auditService)

	// Setup HTTP router
	router := api.SetupRouter(handler, cfg)

	// Initialize gRPC handler
	grpcHandler := grpcserver.NewAdminHandler(adminService, userService, subService, auditService)

	// Initialize gRPC server
	grpcServer := grpcserver.NewServer(grpcHandler, &cfg.GRPC)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start HTTP server in a goroutine
	go func() {
		fmt.Printf("HTTP server starting on port %d\n", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server failed: %v\n", err)
		}
	}()

	// Start gRPC server in a goroutine
	go func() {
		if err := grpcServer.Start(); err != nil {
			fmt.Printf("gRPC server failed: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down servers...")

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("HTTP server forced to shutdown: %v\n", err)
	}

	// Gracefully stop gRPC server
	grpcServer.GracefulStop()

	// Cancel main context
	cancel()

	fmt.Println("Servers stopped. Exiting.")
}
