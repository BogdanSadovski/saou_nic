package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
	"github.com/real-ass/admin-service/internal/service"
)

// AdminHandler implements the gRPC AdminServiceServer interface.
type AdminHandler struct {
	adminService *service.AdminService
	userService  *service.UserService
	subService   *service.SubscriptionService
	auditService *service.AuditService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	adminService *service.AdminService,
	userService *service.UserService,
	subService *service.SubscriptionService,
	auditService *service.AuditService,
) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
		userService:  userService,
		subService:   subService,
		auditService: auditService,
	}
}

// ==================== User Management gRPC Handlers ====================

// GetUser retrieves a user by ID via gRPC.
func (h *AdminHandler) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := h.userService.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &GetUserResponse{
		User: userToProto(user),
	}, nil
}

// ListUsers lists users with pagination via gRPC.
func (h *AdminHandler) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	query := domain.ListUsersQuery{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		Search:   req.Search,
		SortBy:   req.SortBy,
		Order:    req.Order,
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}

	users, total, err := h.userService.ListUsers(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	protoUsers := make([]*UserProto, 0, len(users))
	for _, u := range users {
		protoUsers = append(protoUsers, userToProto(&u))
	}

	return &ListUsersResponse{
		Users:      protoUsers,
		Total:      total,
		Page:       int32(query.Page),
		PageSize:   int32(query.PageSize),
		TotalPages: int32(service.CalculatePagination(total, query.Page, query.PageSize).TotalPages),
	}, nil
}

// CreateUser creates a new user via gRPC.
func (h *AdminHandler) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	createReq := domain.CreateUserRequest{
		Email:     req.Email,
		Username:  req.Username,
		Password:  req.Password,
		Role:      domain.UserRole(req.Role),
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	// In production, extract admin ID from context/metadata
	adminID := uuid.New()

	user, err := h.userService.CreateUser(ctx, createReq, adminID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &CreateUserResponse{
		User: userToProto(user),
	}, nil
}

// UpdateUser updates an existing user via gRPC.
func (h *AdminHandler) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	updateReq := domain.UpdateUserRequest{}

	if req.Email != "" {
		updateReq.Email = &req.Email
	}
	if req.Username != "" {
		updateReq.Username = &req.Username
	}
	if req.Role != "" {
		role := domain.UserRole(req.Role)
		updateReq.Role = &role
	}
	if req.Status != "" {
		status := domain.UserStatus(req.Status)
		updateReq.Status = &status
	}
	if req.FirstName != "" {
		updateReq.FirstName = &req.FirstName
	}
	if req.LastName != "" {
		updateReq.LastName = &req.LastName
	}

	adminID := uuid.New()

	user, err := h.userService.UpdateUser(ctx, id, updateReq, adminID)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &UpdateUserResponse{
		User: userToProto(user),
	}, nil
}

// DeleteUser deletes a user via gRPC.
func (h *AdminHandler) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	adminID := uuid.New()

	if err := h.userService.DeleteUser(ctx, id, adminID); err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return &DeleteUserResponse{
		Success: true,
	}, nil
}

// ==================== Dashboard gRPC Handlers ====================

// GetDashboardStats returns dashboard statistics via gRPC.
func (h *AdminHandler) GetDashboardStats(ctx context.Context, req *GetDashboardStatsRequest) (*GetDashboardStatsResponse, error) {
	stats, err := h.adminService.GetDashboardStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

	return &GetDashboardStatsResponse{
		TotalUsers:          stats.TotalUsers,
		ActiveUsers:         stats.ActiveUsers,
		ActiveSubscriptions: stats.ActiveSubscriptions,
	}, nil
}

// ==================== Helper Functions ====================

// userToProto converts a domain User to a protobuf User message.
func userToProto(user *domain.User) *UserProto {
	if user == nil {
		return nil
	}

	return &UserProto{
		UserId:        user.ID.String(),
		Email:         user.Email,
		Username:      user.Username,
		Role:          string(user.Role),
		Status:        string(user.Status),
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ==================== Proto Message Stubs ====================
// In production, these would be generated from .proto files.

// UserProto represents a user in protobuf format.
type UserProto struct {
	UserId        string `json:"user_id"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// GetUserRequest represents a gRPC request to get a user.
type GetUserRequest struct {
	UserId string `json:"user_id"`
}

// GetUserResponse represents a gRPC response with user data.
type GetUserResponse struct {
	User *UserProto `json:"user"`
}

// ListUsersRequest represents a gRPC request to list users.
type ListUsersRequest struct {
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Search   string `json:"search"`
	SortBy   string `json:"sort_by"`
	Order    string `json:"order"`
}

// ListUsersResponse represents a gRPC response with paginated users.
type ListUsersResponse struct {
	Users      []*UserProto `json:"users"`
	Total      int64        `json:"total"`
	Page       int32        `json:"page"`
	PageSize   int32        `json:"page_size"`
	TotalPages int32        `json:"total_pages"`
}

// CreateUserRequest represents a gRPC request to create a user.
type CreateUserRequest struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// CreateUserResponse represents a gRPC response with created user.
type CreateUserResponse struct {
	User *UserProto `json:"user"`
}

// UpdateUserRequest represents a gRPC request to update a user.
type UpdateUserRequest struct {
	UserId    string `json:"user_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UpdateUserResponse represents a gRPC response with updated user.
type UpdateUserResponse struct {
	User *UserProto `json:"user"`
}

// DeleteUserRequest represents a gRPC request to delete a user.
type DeleteUserRequest struct {
	UserId string `json:"user_id"`
}

// DeleteUserResponse represents a gRPC response for user deletion.
type DeleteUserResponse struct {
	Success bool `json:"success"`
}

// GetDashboardStatsRequest represents a gRPC request for dashboard stats.
type GetDashboardStatsRequest struct{}

// GetDashboardStatsResponse represents a gRPC response with dashboard stats.
type GetDashboardStatsResponse struct {
	TotalUsers          int64 `json:"total_users"`
	ActiveUsers         int64 `json:"active_users"`
	ActiveSubscriptions int64 `json:"active_subscriptions"`
}
