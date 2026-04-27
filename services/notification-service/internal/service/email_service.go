package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hr-automation/notification-service/internal/config"
	"github.com/hr-automation/notification-service/internal/domain"

	"github.com/sirupsen/logrus"
)

// EmailService handles email notification logic with template support.
type EmailService struct {
	sender EmailSender
	repo   domain.NotificationRepository
	config config.EmailConfig
}

// NewEmailService creates a new email service.
func NewEmailService(sender EmailSender, repo domain.NotificationRepository, cfg config.EmailConfig) *EmailService {
	return &EmailService{
		sender: sender,
		repo:   repo,
		config: cfg,
	}
}

// SendWelcomeEmail sends a welcome email to a new user.
func (s *EmailService) SendWelcomeEmail(ctx context.Context, userID int64, email, userName string) error {
	templateData := map[string]any{
		"UserName":  userName,
		"LoginDate": time.Now().Format("January 2, 2006"),
	}

	notification := domain.CreateNotificationRequest{
		UserID:   userID,
		Type:     domain.NotificationTypeWelcome,
		Channel:  domain.NotificationTypeEmail,
		Priority: domain.PriorityNormal,
		Subject:  "Welcome to HR Automation Platform!",
		Body:     fmt.Sprintf("Welcome, %s!", userName),
		Recipient: email,
		Metadata: templateData,
	}

	// Store the notification record
	notif := &domain.Notification{
		UserID:     userID,
		Type:       domain.NotificationTypeWelcome,
		Channel:    domain.NotificationTypeEmail,
		Priority:   domain.PriorityNormal,
		Status:     domain.StatusPending,
		Subject:    notification.Subject,
		Body:       notification.Body,
		Recipient:  email,
		MaxRetries: s.config.MaxRetries,
	}

	if err := s.repo.Create(ctx, notif); err != nil {
		return fmt.Errorf("failed to create welcome notification record: %w", err)
	}

	// Send via template
	err := s.sender.SendTemplate(ctx, email, notification.Subject, "welcome_email.html", templateData)
	if err != nil {
		logrus.WithError(err).WithField("user_id", userID).Error("Failed to send welcome email")
		_ = s.repo.UpdateStatus(ctx, notif.ID, domain.StatusFailed, err.Error())
		return err
	}

	if err := s.repo.MarkAsSent(ctx, notif.ID); err != nil {
		logrus.WithError(err).Error("Failed to mark welcome notification as sent")
	}

	logrus.WithField("user_id", userID).Info("Welcome email sent successfully")
	return nil
}

// SendInterviewReminder sends an interview reminder email.
func (s *EmailService) SendInterviewReminder(ctx context.Context, userID int64, email string, interviewData map[string]any) error {
	notification := &domain.Notification{
		UserID:     userID,
		Type:       domain.NotificationTypeInterview,
		Channel:    domain.NotificationTypeEmail,
		Priority:   domain.PriorityHigh,
		Status:     domain.StatusPending,
		Subject:    "Upcoming Interview Reminder",
		Body:       fmt.Sprintf("You have an upcoming interview on %v", interviewData["interview_date"]),
		Recipient:  email,
		MaxRetries: s.config.MaxRetries,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create interview reminder notification: %w", err)
	}

	err := s.sender.SendTemplate(ctx, email, notification.Subject, "interview_reminder.html", interviewData)
	if err != nil {
		logrus.WithError(err).WithField("user_id", userID).Error("Failed to send interview reminder email")
		_ = s.repo.UpdateStatus(ctx, notification.ID, domain.StatusFailed, err.Error())
		return err
	}

	if err := s.repo.MarkAsSent(ctx, notification.ID); err != nil {
		logrus.WithError(err).Error("Failed to mark interview reminder as sent")
	}

	logrus.WithField("user_id", userID).Info("Interview reminder email sent successfully")
	return nil
}

// SendReportReadyNotification sends a notification when a report is ready.
func (s *EmailService) SendReportReadyNotification(ctx context.Context, userID int64, email string, reportData map[string]any) error {
	notification := &domain.Notification{
		UserID:     userID,
		Type:       domain.NotificationTypeReportReady,
		Channel:    domain.NotificationTypeEmail,
		Priority:   domain.PriorityNormal,
		Status:     domain.StatusPending,
		Subject:    "Your Report is Ready",
		Body:       fmt.Sprintf("The report '%v' is now available for download", reportData["report_name"]),
		Recipient:  email,
		MaxRetries: s.config.MaxRetries,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create report ready notification: %w", err)
	}

	err := s.sender.SendTemplate(ctx, email, notification.Subject, "report_ready.html", reportData)
	if err != nil {
		logrus.WithError(err).WithField("user_id", userID).Error("Failed to send report ready email")
		_ = s.repo.UpdateStatus(ctx, notification.ID, domain.StatusFailed, err.Error())
		return err
	}

	if err := s.repo.MarkAsSent(ctx, notification.ID); err != nil {
		logrus.WithError(err).Error("Failed to mark report ready notification as sent")
	}

	logrus.WithField("user_id", userID).Info("Report ready email sent successfully")
	return nil
}
