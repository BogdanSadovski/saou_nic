package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/hr-automation/notification-service/internal/domain"
	"github.com/hr-automation/notification-service/internal/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
	repo                domain.NotificationRepository
}

// NewNotificationHandler creates a new notification handler.
func NewNotificationHandler(notificationService *service.NotificationService, repo domain.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		repo:                repo,
	}
}

// CreateNotification handles POST /api/v1/notifications
func (h *NotificationHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == 0 || req.Type == "" || req.Channel == "" || req.Body == "" || req.Recipient == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required fields: user_id, type, channel, body, recipient")
		return
	}

	notification, err := h.notificationService.CreateNotification(r.Context(), req)
	if err != nil {
		logrus.WithError(err).Error("Failed to create notification")
		respondWithError(w, http.StatusInternalServerError, "Failed to create notification")
		return
	}

	respondWithJSON(w, http.StatusCreated, notification)
}

// GetNotification handles GET /api/v1/notifications/{id}
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	notification, err := h.notificationService.GetNotification(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Notification not found")
		return
	}

	respondWithJSON(w, http.StatusOK, notification)
}

// GetUserNotifications handles GET /api/v1/users/{user_id}/notifications
func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseInt(vars["user_id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	response, err := h.notificationService.GetUserNotifications(r.Context(), userID, page, pageSize)
	if err != nil {
		logrus.WithError(err).Error("Failed to get user notifications")
		respondWithError(w, http.StatusInternalServerError, "Failed to get notifications")
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// RetryNotification handles POST /api/v1/notifications/{id}/retry
func (h *NotificationHandler) RetryNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	if err := h.notificationService.RetryNotification(r.Context(), id); err != nil {
		logrus.WithError(err).Error("Failed to retry notification")
		respondWithError(w, http.StatusInternalServerError, "Failed to retry notification")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "retry initiated"})
}

// DeleteNotification handles DELETE /api/v1/notifications/{id}
func (h *NotificationHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		logrus.WithError(err).Error("Failed to delete notification")
		respondWithError(w, http.StatusInternalServerError, "Failed to delete notification")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HealthCheck handles GET /health
func (h *NotificationHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "notification-service",
	})
}

// respondWithJSON writes a JSON response.
func respondWithJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logrus.WithError(err).Error("Failed to encode JSON response")
	}
}

// respondWithError writes an error JSON response.
func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, map[string]string{"error": message})
}
