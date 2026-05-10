package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
)

type UserRepositoryAdapter struct {
	repo *PostgresRepository
}

func NewUserRepositoryAdapter(repo *PostgresRepository) *UserRepositoryAdapter {
	return &UserRepositoryAdapter{repo: repo}
}

func (a *UserRepositoryAdapter) Create(ctx context.Context, user *domain.User) error {
	return a.repo.Create(ctx, user)
}

func (a *UserRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *UserRepositoryAdapter) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return a.repo.GetByEmail(ctx, email)
}

func (a *UserRepositoryAdapter) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return a.repo.GetByUsername(ctx, username)
}

func (a *UserRepositoryAdapter) Update(ctx context.Context, user *domain.User) error {
	return a.repo.Update(ctx, user)
}

func (a *UserRepositoryAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.repo.Delete(ctx, id)
}

func (a *UserRepositoryAdapter) List(ctx context.Context, query domain.ListUsersQuery) ([]domain.User, int64, error) {
	return a.repo.List(ctx, query)
}

func (a *UserRepositoryAdapter) Count(ctx context.Context) (int64, error) {
	return a.repo.Count(ctx)
}

func (a *UserRepositoryAdapter) CountCreatedSince(ctx context.Context, since time.Time) (int64, error) {
	return a.repo.CountCreatedSince(ctx, since)
}

func (a *UserRepositoryAdapter) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	return a.repo.UpdateStatus(ctx, id, status)
}

func (a *UserRepositoryAdapter) UpdateRole(ctx context.Context, id uuid.UUID, role domain.UserRole) error {
	return a.repo.UpdateRole(ctx, id, role)
}

func (a *UserRepositoryAdapter) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return a.repo.UpdateLastLogin(ctx, id)
}

type SubscriptionRepositoryAdapter struct {
	repo *PostgresRepository
}

func NewSubscriptionRepositoryAdapter(repo *PostgresRepository) *SubscriptionRepositoryAdapter {
	return &SubscriptionRepositoryAdapter{repo: repo}
}

func (a *SubscriptionRepositoryAdapter) Create(ctx context.Context, subscription *domain.Subscription) error {
	return a.repo.CreateSubscription(ctx, subscription)
}

func (a *SubscriptionRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return a.repo.GetSubscriptionByID(ctx, id)
}

func (a *SubscriptionRepositoryAdapter) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return a.repo.GetByUserID(ctx, userID)
}

func (a *SubscriptionRepositoryAdapter) Update(ctx context.Context, subscription *domain.Subscription) error {
	return a.repo.UpdateSubscription(ctx, subscription)
}

func (a *SubscriptionRepositoryAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeleteSubscription(ctx, id)
}

func (a *SubscriptionRepositoryAdapter) ListByStatus(ctx context.Context, status domain.SubscriptionStatus) ([]domain.Subscription, error) {
	return a.repo.ListByStatus(ctx, status)
}

func (a *SubscriptionRepositoryAdapter) ListByTier(ctx context.Context, tier domain.SubscriptionTier) ([]domain.Subscription, error) {
	return a.repo.ListByTier(ctx, tier)
}

func (a *SubscriptionRepositoryAdapter) ExpireOldSubscriptions(ctx context.Context) (int64, error) {
	return a.repo.ExpireOldSubscriptions(ctx)
}

type AuditLogRepositoryAdapter struct {
	repo *PostgresRepository
}

func NewAuditLogRepositoryAdapter(repo *PostgresRepository) *AuditLogRepositoryAdapter {
	return &AuditLogRepositoryAdapter{repo: repo}
}

func (a *AuditLogRepositoryAdapter) Create(ctx context.Context, log *domain.AuditLog) error {
	return a.repo.CreateAuditLog(ctx, log)
}

func (a *AuditLogRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	return a.repo.GetAuditLogByID(ctx, id)
}

func (a *AuditLogRepositoryAdapter) List(ctx context.Context, filters domain.AuditLogFilters) ([]domain.AuditLog, int64, error) {
	return a.repo.ListAuditLogs(ctx, filters)
}

func (a *AuditLogRepositoryAdapter) ListByAdminID(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]domain.AuditLog, int64, error) {
	return a.repo.ListAuditLogsByAdminID(ctx, adminID, limit, offset)
}

func (a *AuditLogRepositoryAdapter) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	return a.repo.DeleteAuditLogsOlderThan(ctx, days)
}

type RoleRepositoryAdapter struct {
	repo *PostgresRepository
}

func NewRoleRepositoryAdapter(repo *PostgresRepository) *RoleRepositoryAdapter {
	return &RoleRepositoryAdapter{repo: repo}
}

func (a *RoleRepositoryAdapter) Create(ctx context.Context, role *domain.Role) error {
	return a.repo.CreateRole(ctx, role)
}

func (a *RoleRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return a.repo.GetRoleByID(ctx, id)
}

func (a *RoleRepositoryAdapter) GetByName(ctx context.Context, name domain.UserRole) (*domain.Role, error) {
	return a.repo.GetByName(ctx, name)
}

func (a *RoleRepositoryAdapter) List(ctx context.Context) ([]domain.Role, error) {
	return a.repo.ListRoles(ctx)
}

func (a *RoleRepositoryAdapter) Update(ctx context.Context, role *domain.Role) error {
	return a.repo.UpdateRoleRecord(ctx, role)
}

func (a *RoleRepositoryAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeleteRoleRecord(ctx, id)
}
