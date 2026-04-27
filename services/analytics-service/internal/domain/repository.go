package domain

import (
	"context"
)

// EventRepository defines the interface for event storage operations.
type EventRepository interface {
	// Insert stores a single event.
	Insert(ctx context.Context, event *Event) error
	// BatchInsert stores multiple events in a single operation.
	BatchInsert(ctx context.Context, events []*Event) error
	// GetByID retrieves an event by its ID.
	GetByID(ctx context.Context, id string) (*Event, error)
	// Query retrieves events matching the given filter.
	Query(ctx context.Context, filter QueryFilter) ([]*Event, error)
	// Count returns the number of events matching the filter.
	Count(ctx context.Context, filter QueryFilter) (int64, error)
	// DeleteOld removes events older than the given retention period.
	DeleteOld(ctx context.Context, tenantID string, retentionDays int) (int64, error)
}

// MetricsRepository defines the interface for aggregated metrics storage.
type MetricsRepository interface {
	// UpsertMetrics stores or updates aggregated metrics.
	UpsertMetrics(ctx context.Context, metrics *AggregatedMetrics) error
	// BatchUpsertMetrics stores multiple metric records.
	BatchUpsertMetrics(ctx context.Context, metrics []*AggregatedMetrics) error
	// GetTimeSeries retrieves time series data for a metric.
	GetTimeSeries(ctx context.Context, filter QueryFilter) ([]TimeSeriesPoint, error)
	// GetSummary retrieves a metric summary for a time range.
	GetSummary(ctx context.Context, tenantID, metricName string, from, to string) (*MetricSummary, error)
	// GetByTenantAndWindow retrieves aggregated metrics for a tenant and time window.
	GetByTenantAndWindow(ctx context.Context, tenantID string, start, end string, granularity string) ([]*AggregatedMetrics, error)
}

// FunnelRepository defines the interface for funnel storage and computation.
type FunnelRepository interface {
	// CreateFunnel stores a new funnel definition.
	CreateFunnel(ctx context.Context, funnel *Funnel) error
	// GetFunnel retrieves a funnel by ID.
	GetFunnel(ctx context.Context, id string) (*Funnel, error)
	// ListFunnings retrieves all funnels for a tenant.
	ListFunnels(ctx context.Context, tenantID string) ([]*Funnel, error)
	// UpdateFunnel updates an existing funnel definition.
	UpdateFunnel(ctx context.Context, funnel *Funnel) error
	// DeleteFunnel removes a funnel definition.
	DeleteFunnel(ctx context.Context, id string) error
	// ComputeFunnel computes funnel conversion results for a time range.
	ComputeFunnel(ctx context.Context, funnelID string, from, to string) (*FunnelResult, error)
}

// DashboardRepository defines the interface for dashboard storage.
type DashboardRepository interface {
	// Create stores a new dashboard.
	Create(ctx context.Context, dashboard *Dashboard) error
	// GetByID retrieves a dashboard by ID.
	GetByID(ctx context.Context, id string) (*Dashboard, error)
	// List retrieves all dashboards for a tenant.
	List(ctx context.Context, tenantID string) ([]*Dashboard, error)
	// Update modifies an existing dashboard.
	Update(ctx context.Context, dashboard *Dashboard) error
	// Delete removes a dashboard.
	Delete(ctx context.Context, id string) error
}

// ExportRepository defines the interface for export request storage.
type ExportRepository interface {
	// CreateExport stores a new export request.
	CreateExport(ctx context.Context, req *ExportRequest) error
	// GetExport retrieves an export request by ID.
	GetExport(ctx context.Context, id string) (*ExportRequest, error)
	// ListExports retrieves export requests for a tenant.
	ListExports(ctx context.Context, tenantID string, limit, offset int) ([]*ExportRequest, error)
	// UpdateExport modifies an export request status.
	UpdateExport(ctx context.Context, req *ExportRequest) error
}

// UserRepository defines the interface for user analytics data.
type UserRepository interface {
	// TrackSession records a user session.
	TrackSession(ctx context.Context, userID, sessionID string, duration float64) error
	// GetActiveUsers returns the count of active users in a time window.
	GetActiveUsers(ctx context.Context, tenantID string, from, to string) (int64, error)
	// GetUserSegments returns user segmentation data.
	GetUserSegments(ctx context.Context, tenantID string, from, to string) (map[string]int64, error)
	// GetTopUsers retrieves users with the most activity.
	GetTopUsers(ctx context.Context, tenantID string, from, to string, limit int) ([]map[string]interface{}, error)
}
