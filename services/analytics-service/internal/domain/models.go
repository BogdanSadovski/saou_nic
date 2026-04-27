package domain

import (
	"time"
)

// EventType represents the category of an analytics event.
type EventType string

const (
	EventPageView       EventType = "page_view"
	EventClick          EventType = "click"
	EventFormSubmit     EventType = "form_submit"
	EventVideoPlay      EventType = "video_play"
	EventPurchase       EventType = "purchase"
	EventSignup           EventType = "signup"
	EventError          EventType = "error"
	EventCustom         EventType = "custom"
)

// Event represents a raw analytics event ingested from Kafka.
type Event struct {
	ID           string                 `json:"id"`
	Type         EventType              `json:"type"`
	UserID       string                 `json:"user_id,omitempty"`
	SessionID    string                 `json:"session_id"`
	TenantID     string                 `json:"tenant_id"`
	URL          string                 `json:"url,omitempty"`
	Referrer     string                 `json:"referrer,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	IP           string                 `json:"ip,omitempty"`
	Country      string                 `json:"country,omitempty"`
	City         string                 `json:"city,omitempty"`
	Device       string                 `json:"device,omitempty"`
	OS           string                 `json:"os,omitempty"`
	Browser      string                 `json:"browser,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	ProcessedAt  time.Time              `json:"processed_at,omitempty"`
}

// AggregatedMetrics represents pre-computed metrics for a time window.
type AggregatedMetrics struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	WindowStart   time.Time `json:"window_start"`
	WindowEnd     time.Time `json:"window_end"`
	Granularity   string    `json:"granularity"` // minute, hour, day
	TotalEvents   int64     `json:"total_events"`
	UniqueUsers   int64     `json:"unique_users"`
	UniqueSessions int64    `json:"unique_sessions"`
	PageViews     int64     `json:"page_views"`
	Clicks        int64     `json:"clicks"`
	Conversions   int64     `json:"conversions"`
	AvgSessionDuration float64 `json:"avg_session_duration"`
	BounceRate    float64   `json:"bounce_rate"`
	Errors        int64     `json:"errors"`
	CreatedAt     time.Time `json:"created_at"`
}

// FunnelStep represents a single step in a conversion funnel.
type FunnelStep struct {
	StepNumber int       `json:"step_number"`
	Name       string    `json:"name"`
	EventTypes []EventType `json:"event_types"`
	URLPattern string    `json:"url_pattern,omitempty"`
}

// Funnel represents a defined conversion funnel.
type Funnel struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	TenantID  string       `json:"tenant_id"`
	Steps     []FunnelStep `json:"steps"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// FunnelResult holds the computed results for a funnel query.
type FunnelResult struct {
	FunnelID    string         `json:"funnel_id"`
	TotalStart  int64          `json:"total_start"`
	Steps       []FunnelStepResult `json:"steps"`
	ComputedAt  time.Time      `json:"computed_at"`
}

// FunnelStepResult holds conversion data for a single funnel step.
type FunnelStepResult struct {
	StepNumber  int     `json:"step_number"`
	Name        string  `json:"name"`
	Count       int64   `json:"count"`
	ConversionRate float64 `json:"conversion_rate"`
	DropOffRate float64  `json:"drop_off_rate"`
}

// Dashboard represents a saved analytics dashboard configuration.
type Dashboard struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TenantID    string    `json:"tenant_id"`
	Description string    `json:"description,omitempty"`
	Widgets     []Widget  `json:"widgets"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Widget represents a single visualization widget on a dashboard.
type Widget struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Type       string                 `json:"type"` // chart, table, metric, heatmap
	Config     map[string]interface{} `json:"config"`
	Position   WidgetPosition         `json:"position"`
	Size       WidgetSize             `json:"size"`
}

// WidgetPosition holds the grid position of a widget.
type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WidgetSize holds the dimensions of a widget.
type WidgetSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// TimeSeriesPoint represents a single data point in a time series.
type TimeSeriesPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
}

// MetricSummary represents a computed metric with statistical data.
type MetricSummary struct {
	Name   string  `json:"name"`
	Current float64 `json:"current"`
	Previous float64 `json:"previous"`
	Change  float64 `json:"change_percent"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Avg     float64 `json:"avg"`
	P50     float64 `json:"p50"`
	P95     float64 `json:"p95"`
	P99     float64 `json:"p99"`
}

// QueryFilter represents filtering parameters for analytics queries.
type QueryFilter struct {
	TenantID   string     `json:"tenant_id"`
	EventTypes []EventType `json:"event_types,omitempty"`
	UserID     string     `json:"user_id,omitempty"`
	SessionID  string     `json:"session_id,omitempty"`
	URL        string     `json:"url,omitempty"`
	Country    string     `json:"country,omitempty"`
	Device     string     `json:"device,omitempty"`
	OS         string     `json:"os,omitempty"`
	Browser    string     `json:"browser,omitempty"`
	DateFrom   time.Time  `json:"date_from"`
	DateTo     time.Time  `json:"date_to"`
	Granularity string    `json:"granularity,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// ExportRequest represents a request to export analytics data.
type ExportRequest struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Format    string    `json:"format"` // csv, json, xlsx
	Filter    QueryFilter `json:"filter"`
	Status    string    `json:"status"` // pending, processing, completed, failed
	FileURL   string    `json:"file_url,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
