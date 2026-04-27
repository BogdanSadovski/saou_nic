package service

import (
	"context"
	"fmt"
	"time"

	"analytics-service/internal/domain"
	"analytics-service/pkg/aggregator"
)

// AnalyticsService provides core analytics operations.
type AnalyticsService struct {
	eventRepo    domain.EventRepository
	metricsRepo  domain.MetricsRepository
	userRepo     domain.UserRepository
	aggregator   *aggregator.MetricsAggregator
	aggInterval  time.Duration
	retentionDays int
}

// NewAnalyticsService creates a new AnalyticsService.
func NewAnalyticsService(
	eventRepo domain.EventRepository,
	metricsRepo domain.MetricsRepository,
	userRepo domain.UserRepository,
	aggInterval time.Duration,
	retentionDays int,
) *AnalyticsService {
	return &AnalyticsService{
		eventRepo:     eventRepo,
		metricsRepo:   metricsRepo,
		userRepo:      userRepo,
		aggregator:    aggregator.NewMetricsAggregator(),
		aggInterval:   aggInterval,
		retentionDays: retentionDays,
	}
}

// ProcessEvent handles a single incoming event, enriching and storing it.
func (s *AnalyticsService) ProcessEvent(ctx context.Context, event *domain.Event) error {
	event.ProcessedAt = time.Now().UTC()

	if err := s.eventRepo.Insert(ctx, event); err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// ProcessEvents handles a batch of events.
func (s *AnalyticsService) ProcessEvents(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	now := time.Now().UTC()
	for _, e := range events {
		e.ProcessedAt = now
	}

	if err := s.eventRepo.BatchInsert(ctx, events); err != nil {
		return fmt.Errorf("failed to batch insert events: %w", err)
	}

	return nil
}

// GetEvents retrieves events matching the filter.
func (s *AnalyticsService) GetEvents(ctx context.Context, filter domain.QueryFilter) ([]*domain.Event, int64, error) {
	events, err := s.eventRepo.Query(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}

	total, err := s.eventRepo.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	return events, total, nil
}

// GetTimeSeries retrieves time series metrics for a tenant.
func (s *AnalyticsService) GetTimeSeries(
	ctx context.Context,
	filter domain.QueryFilter,
) ([]domain.TimeSeriesPoint, error) {
	points, err := s.metricsRepo.GetTimeSeries(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}

	return points, nil
}

// GetMetricsSummary returns a summary of key metrics for a time range.
func (s *AnalyticsService) GetMetricsSummary(
	ctx context.Context,
	tenantID string,
	from, to string,
) (map[string]*domain.MetricSummary, error) {
	metrics := make(map[string]*domain.MetricSummary)

	keyMetrics := []string{
		"page_views",
		"unique_users",
		"conversions",
		"avg_session_duration",
		"bounce_rate",
	}

	for _, name := range keyMetrics {
		summary, err := s.metricsRepo.GetSummary(ctx, tenantID, name, from, to)
		if err != nil {
			return nil, fmt.Errorf("failed to get summary for %s: %w", name, err)
		}
		if summary != nil {
			metrics[name] = summary
		}
	}

	return metrics, nil
}

// GetActiveUsers returns the count of active users in a time window.
func (s *AnalyticsService) GetActiveUsers(
	ctx context.Context,
	tenantID string,
	from, to string,
) (int64, error) {
	count, err := s.userRepo.GetActiveUsers(ctx, tenantID, from, to)
	if err != nil {
		return 0, fmt.Errorf("failed to get active users: %w", err)
	}

	return count, nil
}

// GetTopUsers retrieves users ranked by activity.
func (s *AnalyticsService) GetTopUsers(
	ctx context.Context,
	tenantID string,
	from, to string,
	limit int,
) ([]map[string]interface{}, error) {
	users, err := s.userRepo.GetTopUsers(ctx, tenantID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}

	return users, nil
}

// RunAggregation computes aggregated metrics from raw events for a time window.
func (s *AnalyticsService) RunAggregation(
	ctx context.Context,
	tenantID string,
	windowStart, windowEnd time.Time,
	granularity string,
) error {
	filter := domain.QueryFilter{
		TenantID:  tenantID,
		DateFrom:  windowStart,
		DateTo:    windowEnd,
		Limit:     100000,
	}

	events, err := s.eventRepo.Query(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to fetch events for aggregation: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	aggregated := s.aggregator.Aggregate(events, windowStart, windowEnd, granularity)

	for _, m := range aggregated {
		if err := s.metricsRepo.UpsertMetrics(ctx, m); err != nil {
			return fmt.Errorf("failed to upsert metrics: %w", err)
		}
	}

	return nil
}

// CleanupOldData removes events older than the retention period.
func (s *AnalyticsService) CleanupOldData(ctx context.Context, tenantID string) (int64, error) {
	deleted, err := s.eventRepo.DeleteOld(ctx, tenantID, s.retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old data: %w", err)
	}

	return deleted, nil
}

// GetRealtimeMetrics returns metrics for the recent real-time window.
func (s *AnalyticsService) GetRealtimeMetrics(
	ctx context.Context,
	tenantID string,
	windowMinutes int,
) (*domain.AggregatedMetrics, error) {
	now := time.Now().UTC()
	start := now.Add(-time.Duration(windowMinutes) * time.Minute)

	metrics, err := s.metricsRepo.GetByTenantAndWindow(
		ctx, tenantID, start.Format(time.RFC3339), now.Format(time.RFC3339), "minute",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get realtime metrics: %w", err)
	}

	if len(metrics) == 0 {
		return &domain.AggregatedMetrics{
			TenantID:    tenantID,
			WindowStart: start,
			WindowEnd:   now,
		}, nil
	}

	// Merge all minute-level metrics into a single summary.
	summary := &domain.AggregatedMetrics{
		TenantID:    tenantID,
		WindowStart: start,
		WindowEnd:   now,
	}

	for _, m := range metrics {
		summary.TotalEvents += m.TotalEvents
		summary.UniqueUsers += m.UniqueUsers
		summary.UniqueSessions += m.UniqueSessions
		summary.PageViews += m.PageViews
		summary.Clicks += m.Clicks
		summary.Conversions += m.Conversions
		summary.Errors += m.Errors
	}

	if len(metrics) > 0 {
		summary.AvgSessionDuration = metrics[0].AvgSessionDuration
		summary.BounceRate = metrics[0].BounceRate
	}

	return summary, nil
}
