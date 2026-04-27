package api

import (
	"net/http"

	"github.com/hr-automation/notification-service/internal/domain"
	"github.com/hr-automation/notification-service/internal/service"

	"github.com/gorilla/mux"
)

// NewRouter creates and configures all HTTP routes.
func NewRouter(notificationService *service.NotificationService, repo domain.NotificationRepository) http.Handler {
	router := mux.NewRouter()

	handler := NewNotificationHandler(notificationService, repo)

	// Health check
	router.HandleFunc("/health", handler.HealthCheck).Methods(http.MethodGet)

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Notification CRUD
	api.HandleFunc("/notifications", handler.CreateNotification).Methods(http.MethodPost)
	api.HandleFunc("/notifications/{id:[0-9]+}", handler.GetNotification).Methods(http.MethodGet)
	api.HandleFunc("/notifications/{id:[0-9]+}/retry", handler.RetryNotification).Methods(http.MethodPost)
	api.HandleFunc("/notifications/{id:[0-9]+}", handler.DeleteNotification).Methods(http.MethodDelete)

	// User notifications
	api.HandleFunc("/users/{user_id:[0-9]+}/notifications", handler.GetUserNotifications).Methods(http.MethodGet)

	// Middleware
	router.Use(loggingMiddleware)
	router.Use(recoveryMiddleware)

	return router
}

// loggingMiddleware logs each request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In production, use structured logging with request ID
		// logrus.WithFields(logrus.Fields{
		//     "method": r.Method,
		//     "path":   r.URL.Path,
		//     "remote": r.RemoteAddr,
		// }).Info("Request received")
		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics and returns 500.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
