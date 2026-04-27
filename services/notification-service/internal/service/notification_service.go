package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hr-automation/notification-service/internal/domain"

	"github.com/sirupsen/logrus"
)

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
	SendTemplate(ctx context.Context, to, subject, templateName string, data map[string]any) error
}

// SMSClient defines the interface for sending SMS messages.
type SMSClient interface {
	Send(ctx context.Context, to, message string) error
}

// PushClient defines the interface for sending push notifications.
type PushClient interface {
	Send(ctx context.Context, deviceToken, title, body string, data map[string]any) error
	SendToTopic(ctx context.Context, topic, title, body string, data map[string]any) error
}

// NotificationService handles notification business logic.
type NotificationService struct {
	repo      domain.NotificationRepository
	email     EmailSender
	sms       SMSClient
	push      PushClient
	maxRetries int
}

// NewNotificationService creates a new notification service.
func NewNotificationService(
	repo domain.NotificationRepository,
	email EmailSender,
	sms SMSClient,
	push PushClient,
	maxRetries int,
) *NotificationService {
	return &NotificationService{
		repo:       repo,
		email:      email,
		sms:        sms,
		push:       push,
		maxRetries: maxRetries,
	}
}

// CreateNotification creates a new notification and dispatches it through the specified channel.
func (s *NotificationService) CreateNotification(ctx context.Context, req domain.CreateNotificationRequest) (*domain.Notification, error) {
	notification := &domain.Notification{
		UserID:     req.UserID,
		Type:       req.Type,
		Channel:    req.Channel,
		Priority:   req.Priority,
		Status:     domain.StatusPending,
		Subject:    req.Subject,
		Body:       req.Body,
		Recipient:  req.Recipient,
		MaxRetries: s.maxRetries,
	}

	if notification.Priority == "" {
		notification.Priority = domain.PriorityNormal
	}

	// Serialize metadata if provided
	if req.Metadata != nil {
		metaJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		notification.Metadata = string(metaJSON)
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Dispatch the notification
	if err := s.dispatch(ctx, notification); err != nil {
		logrus.WithError(err).WithField("notification_id", notification.ID).Error("Failed to dispatch notification")
		_ = s.repo.UpdateStatus(ctx, notification.ID, domain.StatusFailed, err.Error())
	}

	return notification, nil
}

// GetNotification retrieves a notification by ID.
func (s *NotificationService) GetNotification(ctx context.Context, id int64) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// GetUserNotifications retrieves all notifications for a user with pagination.
func (s *NotificationService) GetUserNotifications(ctx context.Context, userID int64, page, pageSize int) (*domain.NotificationListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	notifications, total, err := s.repo.GetByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notifications: %w", err)
	}

	return &domain.NotificationListResponse{
		Notifications: notifications,
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

// ProcessPending processes all pending notifications.
func (s *NotificationService) ProcessPending(ctx context.Context) error {
	notifications, err := s.repo.GetPending(ctx, 50)
	if err != nil {
		return fmt.Errorf("failed to get pending notifications: %w", err)
	}

	for _, n := range notifications {
		if err := s.dispatch(ctx, &n); err != nil {
			logrus.WithError(err).WithField("notification_id", n.ID).Error("Failed to process pending notification")
			s.handleFailure(ctx, n.ID, err)
		}
	}

	return nil
}

// dispatch sends the notification through the appropriate channel.
func (s *NotificationService) dispatch(ctx context.Context, notification *domain.Notification) error {
	switch notification.Channel {
	case domain.NotificationTypeEmail:
		return s.sendEmail(ctx, notification)
	case domain.NotificationTypeSMS:
		return s.sendSMS(ctx, notification)
	case domain.NotificationTypePush:
		return s.sendPush(ctx, notification)
	default:
		return fmt.Errorf("unknown notification channel: %s", notification.Channel)
	}
}

// sendEmail sends a notification via email.
func (s *NotificationService) sendEmail(ctx context.Context, notification *domain.Notification) error {
	logger := logrus.WithField("notification_id", notification.ID).WithField("recipient", notification.Recipient)
	logger.Info("Sending email notification")

	if err := s.email.Send(ctx, notification.Recipient, notification.Subject, notification.Body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if err := s.repo.MarkAsSent(ctx, notification.ID); err != nil {
		logger.WithError(err).Error("Failed to mark notification as sent")
	}

	return nil
}

// sendSMS sends a notification via SMS.
func (s *NotificationService) sendSMS(ctx context.Context, notification *domain.Notification) error {
	logger := logrus.WithField("notification_id", notification.ID).WithField("recipient", notification.Recipient)
	logger.Info("Sending SMS notification")

	if err := s.sms.Send(ctx, notification.Recipient, notification.Body); err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	if err := s.repo.MarkAsSent(ctx, notification.ID); err != nil {
		logger.WithError(err).Error("Failed to mark notification as sent")
	}

	return nil
}

// sendPush sends a notification via push.
func (s *NotificationService) sendPush(ctx context.Context, notification *domain.Notification) error {
	logger := logrus.WithField("notification_id", notification.ID).WithField("recipient", notification.Recipient)
	logger.Info("Sending push notification")

	var data map[string]any
	if notification.Metadata != "" {
		if err := json.Unmarshal([]byte(notification.Metadata), &data); err != nil {
			logger.WithError(err).Warn("Failed to unmarshal metadata")
		}
	}

	if err := s.push.Send(ctx, notification.Recipient, notification.Subject, notification.Body, data); err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	if err := s.repo.MarkAsSent(ctx, notification.ID); err != nil {
		logger.WithError(err).Error("Failed to mark notification as sent")
	}

	return nil
}

// handleFailure handles a failed notification by retrying or marking as failed.
func (s *NotificationService) handleFailure(ctx context.Context, id int64, err error) {
	notification, getErr := s.repo.GetByID(ctx, id)
	if getErr != nil {
		logrus.WithError(getErr).Error("Failed to get notification for retry")
		return
	}

	if notification.RetryCount < notification.MaxRetries {
		if retryErr := s.repo.IncrementRetryCount(ctx, id); retryErr != nil {
			logrus.WithError(retryErr).Error("Failed to increment retry count")
		}
	} else {
		if updateErr := s.repo.UpdateStatus(ctx, id, domain.StatusFailed, err.Error()); updateErr != nil {
			logrus.WithError(updateErr).Error("Failed to update notification status to failed")
		}
	}
}

// RetryNotification attempts to resend a failed notification.
func (s *NotificationService) RetryNotification(ctx context.Context, id int64) error {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if notification.Status != domain.StatusFailed {
		return fmt.Errorf("can only retry failed notifications, current status: %s", notification.Status)
	}

	if err := s.repo.UpdateStatus(ctx, id, domain.StatusPending, ""); err != nil {
		return fmt.Errorf("failed to reset notification status: %w", err)
	}

	return s.dispatch(ctx, notification)
}
