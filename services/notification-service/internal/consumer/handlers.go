package consumer

import (
	"context"
	"fmt"

	"github.com/hr-automation/notification-service/internal/config"
	"github.com/hr-automation/notification-service/internal/domain"
	"github.com/hr-automation/notification-service/internal/service"

	"github.com/sirupsen/logrus"
)

// Handler processes notification messages from RabbitMQ.
type Handler struct {
	notificationService *service.NotificationService
	rabbitConfig        config.RabbitMQConfig
}

// NewHandler creates a new message handler.
func NewHandler(notificationService *service.NotificationService, cfg config.RabbitMQConfig) *Handler {
	return &Handler{
		notificationService: notificationService,
		rabbitConfig:        cfg,
	}
}

// ProcessNotification processes a single notification message from the queue.
func (h *Handler) ProcessNotification(ctx context.Context, data map[string]any) error {
	msg, err := h.parseMessage(data)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id": msg.UserID,
		"type":    msg.Type,
		"channel": msg.Channel,
	}).Info("Processing notification message from queue")

	req := domain.CreateNotificationRequest{
		UserID:   msg.UserID,
		Type:     msg.Type,
		Channel:  msg.Channel,
		Priority: msg.Priority,
		Subject:  msg.Subject,
		Body:     msg.Body,
		Recipient: msg.Recipient,
		Metadata: msg.Metadata,
	}

	if _, err := h.notificationService.CreateNotification(ctx, req); err != nil {
		return fmt.Errorf("failed to create notification from queue message: %w", err)
	}

	return nil
}

func (h *Handler) parseMessage(data map[string]any) (domain.NotificationMessage, error) {
	var msg domain.NotificationMessage

	// Parse user_id
	if userID, ok := data["user_id"].(float64); ok {
		msg.UserID = int64(userID)
	} else {
		return msg, fmt.Errorf("missing or invalid user_id")
	}

	// Parse type
	if t, ok := data["type"].(string); ok {
		msg.Type = domain.NotificationType(t)
	} else {
		return msg, fmt.Errorf("missing or invalid type")
	}

	// Parse channel
	if ch, ok := data["channel"].(string); ok {
		msg.Channel = domain.NotificationType(ch)
	} else {
		// Default to email if channel not specified
		msg.Channel = domain.NotificationTypeEmail
	}

	// Parse priority
	if p, ok := data["priority"].(string); ok {
		msg.Priority = domain.NotificationPriority(p)
	} else {
		msg.Priority = domain.PriorityNormal
	}

	// Parse string fields
	if subject, ok := data["subject"].(string); ok {
		msg.Subject = subject
	}
	if body, ok := data["body"].(string); ok {
		msg.Body = body
	} else {
		return msg, fmt.Errorf("missing or invalid body")
	}
	if recipient, ok := data["recipient"].(string); ok {
		msg.Recipient = recipient
	} else {
		return msg, fmt.Errorf("missing or invalid recipient")
	}

	// Parse metadata if present
	if meta, ok := data["metadata"].(map[string]any); ok {
		msg.Metadata = meta
	}

	return msg, nil
}
