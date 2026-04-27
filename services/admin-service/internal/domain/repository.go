package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, query ListUsersQuery) ([]User, int64, error)
	Count(ctx context.Context) (int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status UserStatus) error
	UpdateRole(ctx context.Context, id uuid.UUID, role UserRole) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// SubscriptionRepository defines the interface for subscription data operations.
type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*Subscription, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	Update(ctx context.Context, subscription *Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByStatus(ctx context.Context, status SubscriptionStatus) ([]Subscription, error)
	ListByTier(ctx context.Context, tier SubscriptionTier) ([]Subscription, error)
	ExpireOldSubscriptions(ctx context.Context) (int64, error)
}

// AuditLogRepository defines the interface for audit log data operations.
type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*AuditLog, error)
	List(ctx context.Context, filters AuditLogFilters) ([]AuditLog, int64, error)
	ListByAdminID(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]AuditLog, int64, error)
	DeleteOlderThan(ctx context.Context, days int) (int64, error)
}

// RoleRepository defines the interface for role data operations.
type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByName(ctx context.Context, name UserRole) (*Role, error)
	List(ctx context.Context) ([]Role, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AuditLogFilters represents filters for querying audit logs.
type AuditLogFilters struct {
	AdminID      *uuid.UUID   `json:"admin_id,omitempty"`
	Action       *AuditAction `json:"action,omitempty"`
	ResourceType string       `json:"resource_type,omitempty"`
	StartDate    *string      `json:"start_date,omitempty"`
	EndDate      *string      `json:"end_date,omitempty"`
	Page         int          `json:"page"`
	PageSize     int          `json:"page_size"`
}
