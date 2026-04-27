package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
)

// UserService handles user management operations.
type UserService struct {
	userRepo  domain.UserRepository
	auditRepo domain.AuditLogRepository
}

// NewUserService creates a new UserService.
func NewUserService(userRepo domain.UserRepository, auditRepo domain.AuditLogRepository) *UserService {
	return &UserService{
		userRepo:  userRepo,
		auditRepo: auditRepo,
	}
}

// CreateUser creates a new user in the system.
func (s *UserService) CreateUser(ctx context.Context, req domain.CreateUserRequest, adminID uuid.UUID) (*domain.User, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Check if username already exists
	existingUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("username %s already exists", req.Username)
	}

	now := time.Now()
	user := &domain.User{
		ID:              uuid.New(),
		Email:           req.Email,
		Username:        req.Username,
		PasswordHash:    hashPassword(req.Password), // In production, use bcrypt
		Role:            req.Role,
		Status:          domain.StatusActive,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		EmailVerified:   false,
		TwoFactorEnabled: false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log the action
	s.logAudit(ctx, adminID, user.Email, domain.ActionCreate, "user", &user.ID,
		fmt.Sprintf("Created user %s with role %s", user.Email, user.Role), "")

	// Don't return password hash
	user.PasswordHash = ""
	return user, nil
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Don't return password hash
	user.PasswordHash = ""
	return user, nil
}

// GetUserByEmail retrieves a user by email.
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	user.PasswordHash = ""
	return user, nil
}

// UpdateUser updates an existing user.
func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, req domain.UpdateUserRequest, adminID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Log the action
	details := fmt.Sprintf("Updated user %s", user.Email)
	s.logAudit(ctx, adminID, user.Email, domain.ActionUpdate, "user", &user.ID, details, "")

	user.PasswordHash = ""
	return user, nil
}

// DeleteUser soft-deletes a user.
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Log the action
	s.logAudit(ctx, adminID, user.Email, domain.ActionDelete, "user", &user.ID,
		fmt.Sprintf("Deleted user %s", user.Email), "")

	return nil
}

// ListUsers retrieves a paginated list of users.
func (s *UserService) ListUsers(ctx context.Context, query domain.ListUsersQuery) ([]domain.User, int64, error) {
	users, total, err := s.userRepo.List(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Clear password hashes
	for i := range users {
		users[i].PasswordHash = ""
	}

	return users, total, nil
}

// SuspendUser suspends a user account.
func (s *UserService) SuspendUser(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.userRepo.UpdateStatus(ctx, id, domain.StatusSuspended); err != nil {
		return fmt.Errorf("failed to suspend user: %w", err)
	}

	s.logAudit(ctx, adminID, user.Email, domain.ActionSuspendUser, "user", &user.ID,
		fmt.Sprintf("Suspended user %s", user.Email), "")

	return nil
}

// BanUser bans a user account.
func (s *UserService) BanUser(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.userRepo.UpdateStatus(ctx, id, domain.StatusBanned); err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}

	s.logAudit(ctx, adminID, user.Email, domain.ActionBanUser, "user", &user.ID,
		fmt.Sprintf("Banned user %s", user.Email), "")

	return nil
}

// ChangeUserRole changes a user's role.
func (s *UserService) ChangeUserRole(ctx context.Context, id uuid.UUID, newRole domain.UserRole, adminID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.userRepo.UpdateRole(ctx, id, newRole); err != nil {
		return fmt.Errorf("failed to change user role: %w", err)
	}

	s.logAudit(ctx, adminID, user.Email, domain.ActionChangeRole, "user", &user.ID,
		fmt.Sprintf("Changed role for user %s from %s to %s", user.Email, user.Role, newRole), "")

	return nil
}

// ActivateUser activates a suspended or inactive user.
func (s *UserService) ActivateUser(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.userRepo.UpdateStatus(ctx, id, domain.StatusActive); err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	s.logAudit(ctx, adminID, user.Email, domain.ActionUpdate, "user", &user.ID,
		fmt.Sprintf("Activated user %s", user.Email), "")

	return nil
}

func (s *UserService) logAudit(ctx context.Context, adminID uuid.UUID, adminEmail string,
	action domain.AuditAction, resourceType string, resourceID *uuid.UUID, details, ipAddress string) {

	log := &domain.AuditLog{
		ID:           uuid.New(),
		AdminID:      adminID,
		AdminEmail:   adminEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		CreatedAt:    time.Now(),
	}

	// Fire and forget - audit logging should not block the main operation
	go func() {
		// In production, use a proper async mechanism or message queue
		_ = s.auditRepo.Create(context.Background(), log)
	}()
}

// hashPassword hashes a plaintext password (stub - use bcrypt in production).
func hashPassword(password string) string {
	// In production, use golang.org/x/crypto/bcrypt
	// hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return "$2a$10$" + password // Stub
}
