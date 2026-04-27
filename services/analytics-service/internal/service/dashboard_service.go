package service

import (
	"context"
	"fmt"
	"time"

	"analytics-service/internal/domain"
)

// DashboardService manages dashboard CRUD and data population.
type DashboardService struct {
	dashboardRepo domain.DashboardRepository
	metricsRepo   domain.MetricsRepository
	eventRepo     domain.EventRepository
	cacheTTL      time.Duration
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(
	dashboardRepo domain.DashboardRepository,
	metricsRepo domain.MetricsRepository,
	eventRepo domain.EventRepository,
	cacheTTL time.Duration,
) *DashboardService {
	return &DashboardService{
		dashboardRepo: dashboardRepo,
		metricsRepo:   metricsRepo,
		eventRepo:     eventRepo,
		cacheTTL:      cacheTTL,
	}
}

// CreateDashboard stores a new dashboard.
func (s *DashboardService) CreateDashboard(
	ctx context.Context,
	dashboard *domain.Dashboard,
) (*domain.Dashboard, error) {
	now := time.Now().UTC()
	dashboard.CreatedAt = now
	dashboard.UpdatedAt = now

	if err := s.dashboardRepo.Create(ctx, dashboard); err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	return dashboard, nil
}

// GetDashboard retrieves a dashboard by ID and populates widget data.
func (s *DashboardService) GetDashboard(
	ctx context.Context,
	id string,
) (*domain.Dashboard, error) {
	dashboard, err := s.dashboardRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	return dashboard, nil
}

// ListDashboards returns all dashboards for a tenant.
func (s *DashboardService) ListDashboards(
	ctx context.Context,
	tenantID string,
) ([]*domain.Dashboard, error) {
	dashboards, err := s.dashboardRepo.List(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list dashboards: %w", err)
	}

	return dashboards, nil
}

// UpdateDashboard modifies an existing dashboard.
func (s *DashboardService) UpdateDashboard(
	ctx context.Context,
	dashboard *domain.Dashboard,
) (*domain.Dashboard, error) {
	dashboard.UpdatedAt = time.Now().UTC()

	if err := s.dashboardRepo.Update(ctx, dashboard); err != nil {
		return nil, fmt.Errorf("failed to update dashboard: %w", err)
	}

	return dashboard, nil
}

// DeleteDashboard removes a dashboard.
func (s *DashboardService) DeleteDashboard(ctx context.Context, id string) error {
	if err := s.dashboardRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	return nil
}

// GetWidgetData fetches the data for a specific widget based on its configuration.
func (s *DashboardService) GetWidgetData(
	ctx context.Context,
	widget domain.Widget,
	tenantID string,
	dateFrom, dateTo string,
) (interface{}, error) {
	widgetType := widget.Type

	switch widgetType {
	case "metric":
		return s.getMetricWidgetData(ctx, widget, tenantID, dateFrom, dateTo)
	case "chart":
		return s.getChartWidgetData(ctx, widget, tenantID, dateFrom, dateTo)
	case "table":
		return s.getTableWidgetData(ctx, widget, tenantID, dateFrom, dateTo)
	default:
		return nil, fmt.Errorf("unsupported widget type: %s", widgetType)
	}
}

func (s *DashboardService) getMetricWidgetData(
	ctx context.Context,
	widget domain.Widget,
	tenantID string,
	dateFrom, dateTo string,
) (*domain.MetricSummary, error) {
	metricName, ok := widget.Config["metric"].(string)
	if !ok {
		return nil, fmt.Errorf("widget config missing metric name")
	}

	summary, err := s.metricsRepo.GetSummary(ctx, tenantID, metricName, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric summary: %w", err)
	}

	return summary, nil
}

func (s *DashboardService) getChartWidgetData(
	ctx context.Context,
	widget domain.Widget,
	tenantID string,
	dateFrom, dateTo string,
) ([]domain.TimeSeriesPoint, error) {
	granularity, ok := widget.Config["granularity"].(string)
	if !ok {
		granularity = "hour"
	}

	filter := domain.QueryFilter{
		TenantID:    tenantID,
		DateFrom:    parseTime(dateFrom),
		DateTo:      parseTime(dateTo),
		Granularity: granularity,
	}

	points, err := s.metricsRepo.GetTimeSeries(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}

	return points, nil
}

func (s *DashboardService) getTableWidgetData(
	ctx context.Context,
	widget domain.Widget,
	tenantID string,
	dateFrom, dateTo string,
) ([]map[string]interface{}, error) {
	limit := 100
	if l, ok := widget.Config["limit"].(float64); ok {
		limit = int(l)
	}

	filter := domain.QueryFilter{
		TenantID: tenantID,
		DateFrom: parseTime(dateFrom),
		DateTo:   parseTime(dateTo),
		Limit:    limit,
	}

	events, err := s.eventRepo.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(events))
	for _, e := range events {
		result = append(result, map[string]interface{}{
			"type":      e.Type,
			"user_id":   e.UserID,
			"url":       e.URL,
			"timestamp": e.Timestamp,
			"country":   e.Country,
			"device":    e.Device,
		})
	}

	return result, nil
}

// GetCacheTTL returns the dashboard cache time-to-live.
func (s *DashboardService) GetCacheTTL() time.Duration {
	return s.cacheTTL
}

func parseTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return time.Now().UTC()
}
