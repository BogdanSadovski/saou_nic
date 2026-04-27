package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bogdan/real_ass/report-service/internal/api"
	"github.com/bogdan/real_ass/report-service/internal/config"
	"github.com/bogdan/real_ass/report-service/internal/repository"
	"github.com/bogdan/real_ass/report-service/internal/service"
	"github.com/bogdan/real_ass/report-service/pkg/generator"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize PostgreSQL repository
	reportRepo, err := repository.NewPostgresRepository(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL repository: %v", err)
	}
	defer reportRepo.Close()

	// Initialize S3 storage
	s3Storage, err := repository.NewS3Storage(cfg.S3)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	// Initialize generators
	pdfGenerator, err := generator.NewPDFGenerator(cfg.Generator)
	if err != nil {
		log.Fatalf("Failed to initialize PDF generator: %v", err)
	}

	docxGenerator, err := generator.NewDOCXGenerator(cfg.Generator)
	if err != nil {
		log.Fatalf("Failed to initialize DOCX generator: %v", err)
	}

	// Initialize services
	reportService := service.NewReportService(reportRepo, s3Storage, cfg.Service)
	pdfGenService := service.NewPDFGeneratorService(pdfGenerator, reportRepo, s3Storage)
	docxGenService := service.NewDOCXGeneratorService(docxGenerator, reportRepo, s3Storage)

	// Initialize handlers and routes
	handlers := api.NewHandlers(reportService, pdfGenService, docxGenService)
	router := api.SetupRoutes(handlers)

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
		log.Printf("Starting report-service on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
