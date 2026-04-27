package domain

import (
	"time"
)

// NotificationType represents the type of notification.
type NotificationType string

const (
	NotificationTypeEmail        NotificationType = "email"
	NotificationTypeSMS          NotificationType = "sms"
	NotificationTypePush         NotificationType = "push"
	NotificationTypeWelcome      NotificationType = "welcome"
	NotificationTypeInterview    NotificationType = "interview_reminder"
	NotificationTypeReportReady  NotificationType = "report_ready"
)

// NotificationStatus represents the current status of a notification.
type NotificationStatus string

const (
	StatusPending    NotificationStatus = "pending"
	StatusSent       NotificationStatus = "sent"
	StatusFailed     NotificationStatus = "failed"
	StatusProcessing NotificationStatus = "processing"
	StatusRetrying   NotificationStatus = "retrying"
)

// NotificationPriority defines the priority level for a notification.
type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityUrgent NotificationPriority = "urgent"
)

// Notification represents a notification entity.
type Notification struct {
	ID             int64              `json:"id" db:"id"`
	UserID         int64              `json:"user_id" db:"user_id"`
	Type           NotificationType   `json:"type" db:"type"`
	Channel        NotificationType   `json:"channel" db:"channel"`
	Priority       NotificationPriority `json:"priority" db:"priority"`
	Status         NotificationStatus `json:"status" db:"status"`
	Subject        string             `json:"subject,omitempty" db:"subject"`
	Body           string             `json:"body" db:"body"`
	Recipient      string             `json:"recipient" db:"recipient"`
	Metadata       string             `json:"metadata,omitempty" db:"metadata"`
	RetryCount     int                `json:"retry_count" db:"retry_count"`
	MaxRetries     int                `json:"max_retries" db:"max_retries"`
	ErrorMessage   string             `json:"error_message,omitempty" db:"error_message"`
	SentAt         *time.Time         `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" db:"updated_at"`
}

// CreateNotificationRequest represents the request to create a notification.
type CreateNotificationRequest struct {
	UserID   int64              `json:"user_id" binding:"required"`
	Type     NotificationType   `json:"type" binding:"required"`
	Channel  NotificationType   `json:"channel" binding:"required"`
	Priority NotificationPriority `json:"priority"`
	Subject  string             `json:"subject"`
	Body     string             `json:"body" binding:"required"`
	Recipient string            `json:"recipient" binding:"required"`
	Metadata map[string]any     `json:"metadata,omitempty"`
}

// UpdateNotificationStatusRequest represents a request to update notification status.
type UpdateNotificationStatusRequest struct {
	Status       NotificationStatus `json:"status" binding:"required"`
	ErrorMessage string             `json:"error_message,omitempty"`
}

// NotificationListResponse represents a paginated list of notifications.
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int64          `json:"total"`
	Page          int            `json:"page"`
	PageSize      int            `json:"page_size"`
}

// NotificationMessage represents a message consumed from RabbitMQ.
type NotificationMessage struct {
	UserID    int64              `json:"user_id"`
	Type      NotificationType   `json:"type"`
	Channel   NotificationType   `json:"channel"`
	Priority  NotificationPriority `json:"priority"`
	Subject   string             `json:"subject"`
	Body      string             `json:"body"`
	Recipient string             `json:"recipient"`
	Metadata  map[string]any     `json:"metadata,omitempty"`
}
