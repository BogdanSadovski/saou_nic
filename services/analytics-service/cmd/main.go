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

	"analytics-service/internal/api"
	"analytics-service/internal/config"
	"analytics-service/internal/consumer"
	"analytics-service/internal/repository"
	"analytics-service/internal/service"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("starting analytics-service")

	// Load configuration.
	cfg := config.Load()
	log.Printf("loaded configuration: server=%s, postgres=%s:%d, clickhouse=%s:%d",
		cfg.Server.Addr(),
		cfg.Postgres.Host, cfg.Postgres.Port,
		cfg.ClickHouse.Host, cfg.ClickHouse.Port,
	)

	// Initialize repositories.
	pgRepo, err := repository.NewPostgresRepository(cfg.Postgres)
	if err != nil {
		log.Fatalf("failed to initialize postgres repository: %v", err)
	}
	defer pgRepo.Close()

	chRepo, err := repository.NewClickHouseRepository(cfg.ClickHouse)
	if err != nil {
		log.Fatalf("failed to initialize clickhouse repository: %v", err)
	}
	defer chRepo.Close()

	// Initialize services.
	analyticsSvc := service.NewAnalyticsService(
		chRepo,
		chRepo,
		pgRepo,
		cfg.Service.AggregationInterval,
		cfg.Service.RetentionDays,
	)

	dashboardSvc := service.NewDashboardService(
		repository.NewDashboardAdapter(pgRepo),
		chRepo,
		chRepo,
		cfg.Service.DashboardCacheTTL,
	)

	exportSvc := service.NewExportService(
		pgRepo,
		chRepo,
		cfg.Service.ExportPageSizeMax,
	)

	// Initialize HTTP handler and router.
	handler := api.NewHandler(analyticsSvc, dashboardSvc, exportSvc)
	router := api.NewRouter(handler, cfg.Server)

	// Initialize Kafka consumer.
	eventHandler := consumer.NewEventHandler(
		chRepo,
		chRepo,
		pgRepo,
		100,            // batch size
		10*time.Second, // flush timeout
	)

	kafkaConsumer := consumer.NewKafkaConsumer(cfg.Kafka, eventHandler)

	// Setup context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Kafka consumer in background.
	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			log.Printf("kafka consumer error: %v", err)
		}
	}()

	// Start event handler flush timer.
	go eventHandler.RunFlushTimer(ctx)

	// Start HTTP server.
	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      router.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background.
	go func() {
		log.Printf("http server listening on %s", cfg.Server.Addr())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down analytics-service...")

	// Stop Kafka consumer.
	if err := kafkaConsumer.Stop(); err != nil {
		log.Printf("error stopping kafka consumer: %v", err)
	}

	// Shutdown HTTP server with timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http server shutdown error: %v", err)
	}

	// Cancel context to stop background goroutines.
	cancel()

	log.Println("analytics-service stopped gracefully")
}

// version is set at build time via ldflags.
var version = "dev"

func printVersion() {
	fmt.Printf("analytics-service version: %s\n", version)
}
