package repository

import (
	"context"

	"analytics-service/internal/domain"
)

// DashboardAdapter adapts PostgresRepository to domain.DashboardRepository.
type DashboardAdapter struct {
	repo *PostgresRepository
}

func NewDashboardAdapter(repo *PostgresRepository) *DashboardAdapter {
	return &DashboardAdapter{repo: repo}
}

func (a *DashboardAdapter) Create(ctx context.Context, dashboard *domain.Dashboard) error {
	return a.repo.CreateDashboard(ctx, dashboard)
}

func (a *DashboardAdapter) GetByID(ctx context.Context, id string) (*domain.Dashboard, error) {
	return a.repo.GetDashboard(ctx, id)
}

func (a *DashboardAdapter) List(ctx context.Context, tenantID string) ([]*domain.Dashboard, error) {
	return a.repo.ListDashboards(ctx, tenantID)
}

func (a *DashboardAdapter) Update(ctx context.Context, dashboard *domain.Dashboard) error {
	return a.repo.UpdateDashboard(ctx, dashboard)
}

func (a *DashboardAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.DeleteDashboard(ctx, id)
}
