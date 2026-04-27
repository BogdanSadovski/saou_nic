package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hr-automation/notification-service/internal/config"
	"github.com/hr-automation/notification-service/internal/domain"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type postgresDB struct {
	db *sqlx.DB
}

// NewPostgresDB creates a new PostgreSQL connection and returns the database wrapper.
func NewPostgresDB(cfg config.DatabaseConfig) (*postgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &postgresDB{db: db}, nil
}

func (r *postgresDB) Close() error {
	return r.db.Close()
}

// NotificationRepository implements domain.NotificationRepository.
type NotificationRepository struct {
	db *sqlx.DB
}

// NewNotificationRepository creates a new notification repository instance.
func NewNotificationRepository(db *postgresDB) *NotificationRepository {
	return &NotificationRepository{db: db.db}
}

func (r *NotificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	query := `
		INSERT INTO notifications (
			user_id, type, channel, priority, status, subject, body,
			recipient, metadata, retry_count, max_retries, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id`

	now := time.Now()
	notification.CreatedAt = now
	notification.UpdatedAt = now

	err := r.db.QueryRowxContext(ctx, query,
		notification.UserID,
		notification.Type,
		notification.Channel,
		notification.Priority,
		notification.Status,
		notification.Subject,
		notification.Body,
		notification.Recipient,
		notification.Metadata,
		notification.RetryCount,
		notification.MaxRetries,
		now,
		now,
	).Scan(&notification.ID)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id int64) (*domain.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, priority, status, subject, body,
		       recipient, metadata, retry_count, max_retries, error_message,
		       sent_at, created_at, updated_at
		FROM notifications WHERE id = $1`

	var notification domain.Notification
	err := r.db.GetContext(ctx, &notification, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

func (r *NotificationRepository) GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]domain.Notification, int64, error) {
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	var total int64
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	query := `
		SELECT id, user_id, type, channel, priority, status, subject, body,
		       recipient, metadata, retry_count, max_retries, error_message,
		       sent_at, created_at, updated_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	var notifications []domain.Notification
	err := r.db.SelectContext(ctx, &notifications, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notifications: %w", err)
	}

	return notifications, total, nil
}

func (r *NotificationRepository) GetByStatus(ctx context.Context, status domain.NotificationStatus, limit int) ([]domain.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, priority, status, subject, body,
		       recipient, metadata, retry_count, max_retries, error_message,
		       sent_at, created_at, updated_at
		FROM notifications WHERE status = $1 ORDER BY created_at ASC LIMIT $2`

	var notifications []domain.Notification
	err := r.db.SelectContext(ctx, &notifications, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications by status: %w", err)
	}

	return notifications, nil
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id int64, status domain.NotificationStatus, errorMsg string) error {
	query := `UPDATE notifications SET status = $1, error_message = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, status, errorMsg, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}
	return nil
}

func (r *NotificationRepository) IncrementRetryCount(ctx context.Context, id int64) error {
	query := `UPDATE notifications SET retry_count = retry_count + 1, status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, domain.StatusRetrying, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}
	return nil
}

func (r *NotificationRepository) MarkAsSent(ctx context.Context, id int64) error {
	now := time.Now()
	query := `UPDATE notifications SET status = $1, sent_at = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, domain.StatusSent, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}
	return nil
}

func (r *NotificationRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM notifications WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("notification with id %d not found", id)
	}

	return nil
}

func (r *NotificationRepository) GetPending(ctx context.Context, limit int) ([]domain.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, priority, status, subject, body,
		       recipient, metadata, retry_count, max_retries, error_message,
		       sent_at, created_at, updated_at
		FROM notifications WHERE status = $1 ORDER BY priority DESC, created_at ASC LIMIT $2`

	var notifications []domain.Notification
	err := r.db.SelectContext(ctx, &notifications, query, domain.StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending notifications: %w", err)
	}

	return notifications, nil
}
