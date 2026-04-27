package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusBanned   UserStatus = "banned"
)

type Provider string

const (
	ProviderLocal   Provider = "local"
	ProviderGoogle  Provider = "google"
	ProviderGitHub  Provider = "github"
)

type User struct {
	ID             uuid.UUID  `json:"id"`
	Email          string     `json:"email"`
	Username       string     `json:"username"`
	PasswordHash   string     `json:"-"`
	FirstName      string     `json:"first_name,omitempty"`
	LastName       string     `json:"last_name,omitempty"`
	AvatarURL      string     `json:"avatar_url,omitempty"`
	Role           UserRole   `json:"role"`
	Status         UserStatus `json:"status"`
	Provider       Provider   `json:"provider"`
	ProviderID     string     `json:"provider_id,omitempty"`
	EmailVerified  bool       `json:"email_verified"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Username  *string `json:"username,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type OAuthUserInfo struct {
	ProviderID string
	Email      string
	Username   string
	FirstName  string
	LastName   string
	AvatarURL  string
	Provider   Provider
}
