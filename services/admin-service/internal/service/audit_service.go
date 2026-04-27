package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
)

// AuditService handles audit log operations.
type AuditService struct {
	auditRepo domain.AuditLogRepository
	userRepo  domain.UserRepository
}

// NewAuditService creates a new AuditService.
func NewAuditService(auditRepo domain.AuditLogRepository, userRepo domain.UserRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
		userRepo:  userRepo,
	}
}

// GetAuditLog retrieves an audit log entry by ID.
func (s *AuditService) GetAuditLog(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	log, err := s.auditRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return log, nil
}

// ListAuditLogs retrieves a paginated list of audit logs with optional filters.
func (s *AuditService) ListAuditLogs(ctx context.Context, filters domain.AuditLogFilters) ([]domain.AuditLog, int64, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}
	if filters.PageSize > 100 {
		filters.PageSize = 100
	}

	logs, total, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return logs, total, nil
}

// GetAuditLogsByAdmin retrieves audit logs for a specific admin user.
func (s *AuditService) GetAuditLogsByAdmin(ctx context.Context, adminID uuid.UUID, page, pageSize int) ([]domain.AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	logs, total, err := s.auditRepo.ListByAdminID(ctx, adminID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs by admin: %w", err)
	}

	return logs, total, nil
}

// GetAuditLogsByAction retrieves audit logs filtered by action type.
func (s *AuditService) GetAuditLogsByAction(ctx context.Context, action domain.AuditAction, page, pageSize int) ([]domain.AuditLog, int64, error) {
	filters := domain.AuditLogFilters{
		Action:   &action,
		Page:     page,
		PageSize: pageSize,
	}

	return s.ListAuditLogs(ctx, filters)
}

// GetAuditLogsByResourceType retrieves audit logs filtered by resource type.
func (s *AuditService) GetAuditLogsByResourceType(ctx context.Context, resourceType string, page, pageSize int) ([]domain.AuditLog, int64, error) {
	filters := domain.AuditLogFilters{
		ResourceType: resourceType,
		Page:         page,
		PageSize:     pageSize,
	}

	return s.ListAuditLogs(ctx, filters)
}

// GetAuditLogsByDateRange retrieves audit logs within a date range.
func (s *AuditService) GetAuditLogsByDateRange(ctx context.Context, startDate, endDate time.Time, page, pageSize int) ([]domain.AuditLog, int64, error) {
	startStr := startDate.Format(time.RFC3339)
	endStr := endDate.Format(time.RFC3339)

	filters := domain.AuditLogFilters{
		StartDate: &startStr,
		EndDate:   &endStr,
		Page:      page,
		PageSize:  pageSize,
	}

	return s.ListAuditLogs(ctx, filters)
}

// CleanupOldLogs deletes audit logs older than the specified number of days.
func (s *AuditService) CleanupOldLogs(ctx context.Context, days int, adminID uuid.UUID) (int64, error) {
	deleted, err := s.auditRepo.DeleteOlderThan(ctx, days)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old audit logs: %w", err)
	}

	// Log the cleanup action
	cleanupLog := &domain.AuditLog{
		ID:           uuid.New(),
		AdminID:      adminID,
		AdminEmail:   "system",
		Action:       domain.ActionDelete,
		ResourceType: "audit_logs",
		Details:      fmt.Sprintf("Cleaned up %d audit logs older than %d days", deleted, days),
		CreatedAt:    time.Now(),
	}

	_ = s.auditRepo.Create(context.Background(), cleanupLog)

	return deleted, nil
}

// ExportAuditLogs exports audit logs for the given filters.
func (s *AuditService) ExportAuditLogs(ctx context.Context, filters domain.AuditLogFilters) ([]domain.AuditLog, error) {
	filters.Page = 1
	filters.PageSize = 10000

	logs, _, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to export audit logs: %w", err)
	}

	return logs, nil
}

// GetActivitySummary returns a summary of admin activity.
type ActivitySummary struct {
	AdminID       uuid.UUID                   `json:"admin_id"`
	AdminEmail    string                      `json:"admin_email"`
	TotalActions  int64                       `json:"total_actions"`
	ActionsByType map[domain.AuditAction]int64 `json:"actions_by_type"`
	LastActivity  *time.Time                  `json:"last_activity"`
}

// GetAdminActivitySummary returns activity summary for an admin user.
func (s *AuditService) GetAdminActivitySummary(ctx context.Context, adminID uuid.UUID) (*ActivitySummary, error) {
	// Get admin user info
	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin user: %w", err)
	}

	logs, total, err := s.auditRepo.ListByAdminID(ctx, adminID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin logs: %w", err)
	}

	summary := &ActivitySummary{
		AdminID:       adminID,
		AdminEmail:    admin.Email,
		TotalActions:  total,
		ActionsByType: make(map[domain.AuditAction]int64),
	}

	for _, log := range logs {
		summary.ActionsByType[log.Action]++
		if summary.LastActivity == nil || log.CreatedAt.After(*summary.LastActivity) {
			summary.LastActivity = &log.CreatedAt
		}
	}

	return summary, nil
}
