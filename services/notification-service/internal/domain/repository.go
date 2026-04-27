package domain

import (
	"context"
)

// NotificationRepository defines the interface for notification persistence operations.
type NotificationRepository interface {
	// Create inserts a new notification into the database.
	Create(ctx context.Context, notification *Notification) error

	// GetByID retrieves a notification by its ID.
	GetByID(ctx context.Context, id int64) (*Notification, error)

	// GetByUserID retrieves all notifications for a given user with pagination.
	GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]Notification, int64, error)

	// GetByStatus retrieves notifications filtered by status.
	GetByStatus(ctx context.Context, status NotificationStatus, limit int) ([]Notification, error)

	// UpdateStatus updates the status of a notification.
	UpdateStatus(ctx context.Context, id int64, status NotificationStatus, errorMsg string) error

	// IncrementRetryCount increments the retry count for a notification.
	IncrementRetryCount(ctx context.Context, id int64) error

	// Update marks a notification as sent with the sent timestamp.
	MarkAsSent(ctx context.Context, id int64) error

	// Delete removes a notification from the database.
	Delete(ctx context.Context, id int64) error

	// GetPending retrieves all pending notifications for processing.
	GetPending(ctx context.Context, limit int) ([]Notification, error)
}
