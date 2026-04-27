package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"scoring-service/internal/api"
	"scoring-service/internal/config"
	grpcserver "scoring-service/internal/grpc"
	"scoring-service/internal/repository"
	"scoring-service/internal/service"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Initialize logger
	log.Printf("starting scoring-service with config: %s", *configPath)

	// Initialize database repository
	repo, err := repository.NewPostgresRepository(
		cfg.Database.DSN(),
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer repo.Close()

	scoreRepo := repository.NewScoreRepositoryAdapter(repo)
	rubricRepo := repository.NewRubricRepositoryAdapter(repo)

	// Initialize services
	scoringService := service.NewScoringService(scoreRepo, rubricRepo, cfg.Scoring.DefaultPassThreshold)

	// Initialize HTTP server
	httpHandler := api.NewHandler(scoringService)
	httpServer := &http.Server{
		Addr:         cfg.Server.HTTPAddr(),
		Handler:      httpHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Initialize gRPC server
	grpcHandler := grpcserver.NewScoringHandler(scoringService)
	grpcServer, err := grpcserver.NewServer(cfg.GRPC.GRPCAddr(), grpcHandler)
	if err != nil {
		log.Fatalf("failed to initialize gRPC server: %v", err)
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("HTTP server listening on %s", cfg.Server.HTTPAddr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start gRPC server in a goroutine
	go func() {
		log.Printf("gRPC server listening on %s", cfg.GRPC.GRPCAddr())
		if err := grpcServer.Start(); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down servers...")

	// Gracefully shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	// Gracefully shutdown gRPC server
	grpcServer.GracefulStop()

	log.Println("servers stopped")
}
