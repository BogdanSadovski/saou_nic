package service

import (
	"context"
	"fmt"

	"github.com/hr-automation/notification-service/internal/config"
)

// SMSService handles SMS notification sending.
type SMSService struct {
	config config.SMSConfig
	client SMSClient
}

// NewSMSService creates a new SMS service.
func NewSMSService(cfg config.SMSConfig) *SMSService {
	return &SMSService{
		config: cfg,
	}
}

// Send sends an SMS message to the specified phone number.
func (s *SMSService) Send(ctx context.Context, to, message string) error {
	if s.client == nil {
		// In a real implementation, this would use Twilio SDK or similar
		// For now, this is a placeholder for the SMS sending logic
		return fmt.Errorf("SMS client not initialized")
	}

	if err := s.client.Send(ctx, to, message); err != nil {
		return fmt.Errorf("failed to send SMS to %s: %w", to, err)
	}

	return nil
}

// SendVerificationCode sends a verification code via SMS.
func (s *SMSService) SendVerificationCode(ctx context.Context, phoneNumber, code string) error {
	message := fmt.Sprintf("Your HR Automation verification code is: %s. Do not share this code with anyone.", code)
	return s.Send(ctx, phoneNumber, message)
}

// SendAppointmentReminder sends an appointment reminder via SMS.
func (s *SMSService) SendAppointmentReminder(ctx context.Context, phoneNumber, dateTime string) error {
	message := fmt.Sprintf("Reminder: You have an appointment on %s. Reply C to confirm.", dateTime)
	return s.Send(ctx, phoneNumber, message)
}
