package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"resume-service/internal/api"
	"resume-service/internal/config"
	"resume-service/internal/repository"
	"resume-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize PostgreSQL repository
	repo, err := repository.NewPostgresRepository(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// Initialize S3 storage
	s3Storage, err := repository.NewS3Storage(cfg.S3)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	// Initialize NLP analyzer
	nlpAnalyzer := service.NewNLPAnalyzer()

	// Initialize resume service
	resumeService := service.NewResumeService(repo, s3Storage, nlpAnalyzer)

	// Initialize handlers and routes
	handler := api.NewHandler(resumeService)
	router := api.NewRouter(handler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting resume-service on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
