package service

import (
	"context"
	"fmt"

	"github.com/hr-automation/notification-service/internal/domain"

	"github.com/sirupsen/logrus"
)

// PushService handles push notification logic.
type PushService struct {
	client PushClient
	repo   domain.NotificationRepository
}

// NewPushService creates a new push notification service.
func NewPushService(client PushClient, repo domain.NotificationRepository) *PushService {
	return &PushService{
		client: client,
		repo:   repo,
	}
}

// SendToDevice sends a push notification to a specific device token.
func (s *PushService) SendToDevice(ctx context.Context, userID int64, deviceToken, title, body string, data map[string]any) error {
	notification := &domain.Notification{
		UserID:     userID,
		Type:       domain.NotificationTypePush,
		Channel:    domain.NotificationTypePush,
		Status:     domain.StatusPending,
		Subject:    title,
		Body:       body,
		Recipient:  deviceToken,
		MaxRetries: 3,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create push notification record: %w", err)
	}

	if err := s.client.Send(ctx, deviceToken, title, body, data); err != nil {
		logrus.WithError(err).WithField("user_id", userID).Error("Failed to send push notification")
		_ = s.repo.UpdateStatus(ctx, notification.ID, domain.StatusFailed, err.Error())
		return err
	}

	if err := s.repo.MarkAsSent(ctx, notification.ID); err != nil {
		logrus.WithError(err).Error("Failed to mark push notification as sent")
	}

	logrus.WithField("user_id", userID).Info("Push notification sent successfully")
	return nil
}

// SendToTopic sends a push notification to a Firebase topic.
func (s *PushService) SendToTopic(ctx context.Context, topic, title, body string, data map[string]any) error {
	if err := s.client.SendToTopic(ctx, topic, title, body, data); err != nil {
		return fmt.Errorf("failed to send push notification to topic %s: %w", topic, err)
	}

	logrus.WithField("topic", topic).Info("Push notification sent to topic successfully")
	return nil
}
