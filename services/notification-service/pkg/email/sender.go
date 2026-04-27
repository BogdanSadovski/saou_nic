package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"time"

	"github.com/hr-automation/notification-service/internal/config"

	"github.com/sirupsen/logrus"
)

// Sender handles sending emails via SMTP.
type Sender struct {
	config     config.EmailConfig
	auth       smtp.Auth
	templates  *template.Template
}

// NewSender creates a new email sender.
func NewSender(cfg config.EmailConfig) *Sender {
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)

	sender := &Sender{
		config: cfg,
		auth:   auth,
	}

	// Pre-load templates
	sender.loadTemplates()

	return sender
}

// Send sends a plain text or HTML email.
func (s *Sender) Send(ctx context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	from := fmt.Sprintf("%s <%s>", s.config.FromName, s.config.From)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	)

	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, s.auth, s.config.From, []string{to}, []byte(msg))
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("sending email timed out: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	logrus.WithField("recipient", to).Debug("Email sent successfully")
	return nil
}

// SendTemplate sends an email using a named template with data.
func (s *Sender) SendTemplate(ctx context.Context, to, subject, templateName string, data map[string]any) error {
	if s.templates == nil {
		return fmt.Errorf("no templates loaded")
	}

	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return s.Send(ctx, to, subject, buf.String())
}

// loadTemplates loads all HTML templates from the configured path.
func (s *Sender) loadTemplates() {
	pattern := filepath.Join(s.config.Path, "*.html")
	tmpl, err := template.New("email").ParseGlob(pattern)
	if err != nil {
		logrus.WithError(err).Warn("Failed to load email templates; template sending will be disabled")
		return
	}
	s.templates = tmpl
	logrus.Info("Email templates loaded successfully")
}

// SendWithRetry attempts to send an email with retry logic.
func (s *Sender) SendWithRetry(ctx context.Context, to, subject, body string, maxRetries int) error {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := s.Send(ctx, to, subject, body)
		if err == nil {
			return nil
		}

		lastErr = err
		logrus.WithError(err).WithFields(logrus.Fields{
			"attempt":  attempt,
			"max":      maxRetries,
			"recipient": to,
		}).Warn("Email send failed, retrying...")

		// Exponential backoff
		time.Sleep(time.Duration(attempt*attempt) * time.Second)
	}

	return fmt.Errorf("failed to send email after %d attempts: %w", maxRetries, lastErr)
}
