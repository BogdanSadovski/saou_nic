package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/real-ass/user-service/internal/domain"
	"github.com/real-ass/user-service/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

// Note: In production, embed the generated UnimplementedUserServiceServer
// type UserHandler struct {
//     pb.UnimplementedUserServiceServer
//     userService *service.UserService
// }

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUser retrieves a user by ID
// Note: Signature will change when using actual protobuf-generated code
func (h *UserHandler) GetUser(ctx context.Context, userID string) (interface{}, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// ListUsers retrieves a list of users
// Note: Signature will change when using actual protobuf-generated code
func (h *UserHandler) ListUsers(ctx context.Context, limit, offset int32) (interface{}, error) {
	users, err := h.userService.ListUsers(ctx, int(limit), int(offset))
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// UpdateUser updates a user's information
// Note: Signature will change when using actual protobuf-generated code
func (h *UserHandler) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) (interface{}, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	req := parseUpdateRequest(updates)
	user, err := h.userService.UpdateUser(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// parseUpdateRequest converts a generic map to domain.UpdateUserRequest
// This is a placeholder - actual implementation will use protobuf types
func parseUpdateRequest(updates map[string]interface{}) domain.UpdateUserRequest {
	var req domain.UpdateUserRequest

	if v, ok := updates["first_name"].(string); ok {
		req.FirstName = &v
	}
	if v, ok := updates["last_name"].(string); ok {
		req.LastName = &v
	}
	if v, ok := updates["username"].(string); ok {
		req.Username = &v
	}
	if v, ok := updates["avatar_url"].(string); ok {
		req.AvatarURL = &v
	}

	return req
}
