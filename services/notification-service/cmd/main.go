package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hr-automation/notification-service/internal/api"
	"github.com/hr-automation/notification-service/internal/config"
	"github.com/hr-automation/notification-service/internal/consumer"
	"github.com/hr-automation/notification-service/internal/repository"
	"github.com/hr-automation/notification-service/internal/service"
	"github.com/hr-automation/notification-service/pkg/email"
	"github.com/hr-automation/notification-service/pkg/push"

	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize logger
	initLogger(cfg)

	logrus.Info("Starting notification service")

	// Initialize database
	db, err := repository.NewPostgresDB(cfg.Database)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	logrus.Info("Connected to database")

	// Initialize services
	emailSender := email.NewSender(cfg.Email)
	firebaseApp, err := push.NewFirebaseApp(cfg.Firebase)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize Firebase; push notifications will be disabled")
	}
	pushNotifier := push.NewNotifier(firebaseApp)

	notificationRepo := repository.NewNotificationRepository(db)

	smsService := service.NewSMSService(cfg.SMS)

	notificationService := service.NewNotificationService(
		notificationRepo,
		emailSender,
		smsService,
		pushNotifier,
		cfg.Email.MaxRetries,
	)

	// Initialize RabbitMQ consumer
	consumerHandler := consumer.NewHandler(notificationService, cfg.RabbitMQ)
	rabbitConsumer, err := consumer.NewRabbitMQ(cfg.RabbitMQ, consumerHandler)
	if err != nil {
		logrus.WithError(err).Warn("Failed to connect to RabbitMQ; message consuming will be disabled")
	} else {
		if err := rabbitConsumer.Start(); err != nil {
			logrus.WithError(err).Warn("Failed to start RabbitMQ consumer")
		} else {
			defer rabbitConsumer.Stop()
			logrus.Info("RabbitMQ consumer started")
		}
	}

	// Initialize HTTP server
	router := api.NewRouter(notificationService, notificationRepo)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("HTTP server listening on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Server forced to shutdown")
	}

	logrus.Info("Server exited")
}

func initLogger(cfg *config.Config) {
	switch cfg.Log.Level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	if cfg.Log.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}
