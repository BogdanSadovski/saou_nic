package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the possible roles a user can have in the system.
type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleAdmin      UserRole = "admin"
	RoleModerator  UserRole = "moderator"
	RoleUser       UserRole = "user"
)

// UserStatus represents the current status of a user account.
type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusSuspended UserStatus = "suspended"
	StatusBanned   UserStatus = "banned"
)

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus string

const (
	SubscriptionActive   SubscriptionStatus = "active"
	SubscriptionExpired  SubscriptionStatus = "expired"
	SubscriptionCanceled SubscriptionStatus = "canceled"
	SubscriptionPending  SubscriptionStatus = "pending"
)

// SubscriptionTier represents the subscription plan tier.
type SubscriptionTier string

const (
	TierFree     SubscriptionTier = "free"
	TierBasic    SubscriptionTier = "basic"
	TierPro      SubscriptionTier = "pro"
	TierEnterprise SubscriptionTier = "enterprise"
)

// AuditAction represents the type of audit action performed.
type AuditAction string

const (
	ActionCreate       AuditAction = "create"
	ActionUpdate       AuditAction = "update"
	ActionDelete       AuditAction = "delete"
	ActionLogin        AuditAction = "login"
	ActionLogout       AuditAction = "logout"
	ActionSuspendUser  AuditAction = "suspend_user"
	ActionBanUser      AuditAction = "ban_user"
	ActionChangeRole   AuditAction = "change_role"
	ActionChangeSubscription AuditAction = "change_subscription"
)

// User represents an admin system user.
type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	Username        string     `json:"username"`
	PasswordHash    string     `json:"-"`
	Role            UserRole   `json:"role"`
	Status          UserStatus `json:"status"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	EmailVerified   bool       `json:"email_verified"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// Role represents a system role with associated permissions.
type Role struct {
	ID          uuid.UUID   `json:"id"`
	Name        UserRole    `json:"name"`
	Description string      `json:"description"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Permission represents a single permission in the RBAC system.
type Permission struct {
	ID        uuid.UUID `json:"id"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

// Subscription represents a user's subscription plan.
type Subscription struct {
	ID            uuid.UUID          `json:"id"`
	UserID        uuid.UUID          `json:"user_id"`
	Tier          SubscriptionTier   `json:"tier"`
	Status        SubscriptionStatus `json:"status"`
	StartDate     time.Time          `json:"start_date"`
	EndDate       *time.Time         `json:"end_date,omitempty"`
	AutoRenew     bool               `json:"auto_renew"`
	MaxUsers      int                `json:"max_users"`
	MaxStorageGB  int                `json:"max_storage_gb"`
	Features      []string           `json:"features"`
	// Metadata — произвольный JSONB на стороне БД. Используем any-значения,
	// потому что seed/реальные подписки кладут туда числа (`price`, `amount`),
	// булевы (`auto_renew_disabled`), а не только строки.
	Metadata      map[string]any     `json:"metadata"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// AuditLog represents an audit trail entry for admin actions.
type AuditLog struct {
	ID           uuid.UUID   `json:"id"`
	AdminID      uuid.UUID   `json:"admin_id"`
	AdminEmail   string      `json:"admin_email"`
	Action       AuditAction `json:"action"`
	ResourceType string      `json:"resource_type"`
	ResourceID   *uuid.UUID  `json:"resource_id,omitempty"`
	Details      string      `json:"details"`
	IPAddress    string      `json:"ip_address"`
	UserAgent    string      `json:"user_agent"`
	CreatedAt    time.Time   `json:"created_at"`
}

// CreateUserRequest represents the request to create a new user.
type CreateUserRequest struct {
	Email     string   `json:"email" binding:"required,email"`
	Username  string   `json:"username" binding:"required,min=3,max=50"`
	Password  string   `json:"password" binding:"required,min=8"`
	Role      UserRole `json:"role" binding:"required,oneof=super_admin admin moderator user"`
	FirstName string   `json:"first_name" binding:"max=100"`
	LastName  string   `json:"last_name" binding:"max=100"`
}

// UpdateUserRequest represents the request to update an existing user.
type UpdateUserRequest struct {
	Email     *string    `json:"email" binding:"omitempty,email"`
	Username  *string    `json:"username" binding:"omitempty,min=3,max=50"`
	Role      *UserRole  `json:"role" binding:"omitempty,oneof=super_admin admin moderator user"`
	Status    *UserStatus `json:"status" binding:"omitempty,oneof=active inactive suspended banned"`
	FirstName *string    `json:"first_name" binding:"omitempty,max=100"`
	LastName  *string    `json:"last_name" binding:"omitempty,max=100"`
}

// ListUsersQuery represents query parameters for listing users.
type ListUsersQuery struct {
	Page     int        `form:"page,default=1"`
	PageSize int        `form:"page_size,default=20"`
	Role     *UserRole  `form:"role"`
	Status   *UserStatus `form:"status"`
	Search   string     `form:"search"`
	SortBy   string     `form:"sort_by,default=created_at"`
	Order    string     `form:"order,default=desc"`
}

// PaginatedResponse represents a paginated API response.
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}
