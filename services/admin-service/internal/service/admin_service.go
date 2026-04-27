package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
)

// AdminService provides high-level admin operations.
type AdminService struct {
	userRepo       domain.UserRepository
	subRepo        domain.SubscriptionRepository
	auditRepo      domain.AuditLogRepository
	roleRepo       domain.RoleRepository
}

// NewAdminService creates a new AdminService.
func NewAdminService(
	userRepo domain.UserRepository,
	subRepo domain.SubscriptionRepository,
	auditRepo domain.AuditLogRepository,
	roleRepo domain.RoleRepository,
) *AdminService {
	return &AdminService{
		userRepo:  userRepo,
		subRepo:   subRepo,
		auditRepo: auditRepo,
		roleRepo:  roleRepo,
	}
}

// DashboardStats represents dashboard overview statistics.
type DashboardStats struct {
	TotalUsers           int64                    `json:"total_users"`
	ActiveUsers          int64                    `json:"active_users"`
	NewUsersToday        int64                    `json:"new_users_today"`
	TotalSubscriptions   int64                    `json:"total_subscriptions"`
	ActiveSubscriptions  int64                    `json:"active_subscriptions"`
	RevenueThisMonth     float64                  `json:"revenue_this_month"`
	RecentAuditLogs      []domain.AuditLog        `json:"recent_audit_logs"`
	RoleDistribution     map[domain.UserRole]int64 `json:"role_distribution"`
	SubscriptionTiers    map[domain.SubscriptionTier]int64 `json:"subscription_tiers"`
}

// GetDashboardStats returns overview statistics for the admin dashboard.
func (s *AdminService) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{
		RoleDistribution:  make(map[domain.UserRole]int64),
		SubscriptionTiers: make(map[domain.SubscriptionTier]int64),
	}

	// Total users
	totalUsers, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	stats.TotalUsers = totalUsers

	// Active users (simplified - in production would filter by status)
	users, total, err := s.userRepo.List(ctx, domain.ListUsersQuery{
		Page:     1,
		PageSize: 1000,
		Status:   ptrStatus(domain.StatusActive),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list active users: %w", err)
	}
	stats.ActiveUsers = total

	// Count role distribution
	for _, u := range users {
		stats.RoleDistribution[u.Role]++
	}

	// Get subscriptions
	activeSubs, err := s.subRepo.ListByStatus(ctx, domain.SubscriptionActive)
	if err != nil {
		return nil, fmt.Errorf("failed to list active subscriptions: %w", err)
	}
	stats.ActiveSubscriptions = int64(len(activeSubs))

	// Count subscription tiers
	for _, sub := range activeSubs {
		stats.SubscriptionTiers[sub.Tier]++
	}

	// Recent audit logs
	logs, _, err := s.auditRepo.List(ctx, domain.AuditLogFilters{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	stats.RecentAuditLogs = logs

	return stats, nil
}

// SystemHealth represents system health information.
type SystemHealth struct {
	DatabaseConnected bool      `json:"database_connected"`
	LastChecked       time.Time `json:"last_checked"`
	Uptime            string    `json:"uptime"`
	Version           string    `json:"version"`
}

// GetSystemHealth returns system health status.
func (s *AdminService) GetSystemHealth(ctx context.Context) *SystemHealth {
	health := &SystemHealth{
		LastChecked: time.Now(),
		Version:     "1.0.0",
	}

	// Check database connectivity
	_, err := s.userRepo.Count(ctx)
	health.DatabaseConnected = err == nil

	return health
}

// BulkUpdateStatus performs bulk status update on users.
func (s *AdminService) BulkUpdateStatus(ctx context.Context, userIDs []uuid.UUID, status domain.UserStatus) (int, error) {
	updated := 0

	for _, id := range userIDs {
		if err := s.userRepo.UpdateStatus(ctx, id, status); err != nil {
			// Log error but continue with other users
			continue
		}
		updated++
	}

	return updated, nil
}

// ExportUsers exports user data (simplified implementation).
func (s *AdminService) ExportUsers(ctx context.Context, query domain.ListUsersQuery) ([]domain.User, error) {
	query.Page = 1
	query.PageSize = 10000 // Large page for export

	users, _, err := s.userRepo.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to export users: %w", err)
	}

	return users, nil
}

// CalculatePagination calculates pagination metadata.
func CalculatePagination(total int64, page, pageSize int) domain.PaginatedResponse {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return domain.PaginatedResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

func ptrStatus(s domain.UserStatus) *domain.UserStatus {
	return &s
}
