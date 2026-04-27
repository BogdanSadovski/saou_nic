package push

import (
	"context"
	"fmt"
	"os"

	"github.com/hr-automation/notification-service/internal/config"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// Notifier handles push notification delivery via Firebase Cloud Messaging.
type Notifier struct {
	client *messaging.Client
	app    *firebase.App
}

// NewFirebaseApp initializes the Firebase app for push notifications.
func NewFirebaseApp(cfg config.FirebaseConfig) (*firebase.App, error) {
	ctx := context.Background()

	var opt option.ClientOption
	if cfg.ServiceAccountKey != "" {
		// Check if the file exists
		if _, err := os.Stat(cfg.ServiceAccountKey); err != nil {
			return nil, fmt.Errorf("firebase service account key file not found: %w", err)
		}
		opt = option.WithCredentialsFile(cfg.ServiceAccountKey)
	} else {
		// Use default credentials (e.g., from environment)
		return nil, fmt.Errorf("firebase service account key not configured")
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: cfg.ProjectID,
	}, opt)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	return app, nil
}

// NewNotifier creates a new push notifier from a Firebase app.
func NewNotifier(app *firebase.App) *Notifier {
	if app == nil {
		return &Notifier{}
	}

	ctx := context.Background()
	client, err := app.Messaging(ctx)
	if err != nil {
		logrus.WithError(err).Warn("Failed to get Firebase messaging client")
		return &Notifier{app: app}
	}

	return &Notifier{
		client: client,
		app:    app,
	}
}

// Send sends a push notification to a specific device token.
func (n *Notifier) Send(ctx context.Context, deviceToken, title, body string, data map[string]any) error {
	if n.client == nil {
		return fmt.Errorf("firebase messaging client not initialized")
	}

	message := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: make(map[string]string),
	}

	// Convert data map to string values
	for k, v := range data {
		message.Data[k] = fmt.Sprintf("%v", v)
	}

	response, err := n.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	logrus.WithField("message_id", response).Debug("Push notification sent successfully")
	return nil
}

// SendToTopic sends a push notification to a Firebase topic.
func (n *Notifier) SendToTopic(ctx context.Context, topic, title, body string, data map[string]any) error {
	if n.client == nil {
		return fmt.Errorf("firebase messaging client not initialized")
	}

	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: make(map[string]string),
	}

	// Convert data map to string values
	for k, v := range data {
		message.Data[k] = fmt.Sprintf("%v", v)
	}

	response, err := n.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send push notification to topic %s: %w", topic, err)
	}

	logrus.WithFields(logrus.Fields{
		"topic":      topic,
		"message_id": response,
	}).Debug("Push notification sent to topic successfully")
	return nil
}

// SendMulticast sends a push notification to multiple device tokens.
func (n *Notifier) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]any) (*messaging.BatchResponse, error) {
	if n.client == nil {
		return nil, fmt.Errorf("firebase messaging client not initialized")
	}

	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: make(map[string]string),
	}

	// Convert data map to string values
	for k, v := range data {
		message.Data[k] = fmt.Sprintf("%v", v)
	}

	response, err := n.client.SendMulticast(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send multicast push notification: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"success_count": response.SuccessCount,
		"failure_count": response.FailureCount,
	}).Info("Multicast push notification sent")

	return response, nil
}
