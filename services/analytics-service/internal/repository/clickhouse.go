package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"analytics-service/internal/config"
	"analytics-service/internal/domain"
)

// ClickHouseRepository implements event and metrics storage using ClickHouse.
type ClickHouseRepository struct {
	db *sql.DB
}

// NewClickHouseRepository creates a new ClickHouse repository.
func NewClickHouseRepository(cfg config.ClickHouseConfig) (*ClickHouseRepository, error) {
	// In production, use github.com/ClickHouse/clickhouse-go/v2 driver.
	// Here we use database/sql as the interface; the actual driver would be registered.
	db, err := sql.Open("clickhouse", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return &ClickHouseRepository{db: db}, nil
}

// Close closes the database connection.
func (r *ClickHouseRepository) Close() error {
	return r.db.Close()
}

// EventRepository implementation for ClickHouse.

func (r *ClickHouseRepository) Insert(ctx context.Context, event *domain.Event) error {
	query := `
		INSERT INTO events (
			id, type, user_id, session_id, tenant_id, url, referrer,
			user_agent, ip, country, city, device, os, browser,
			properties, timestamp, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	propsJSON := "{}"
	if len(event.Properties) > 0 {
		b, err := json.Marshal(event.Properties)
		if err != nil {
			return fmt.Errorf("failed to marshal properties: %w", err)
		}
		propsJSON = string(b)
	}

	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.Type, event.UserID, event.SessionID, event.TenantID,
		event.URL, event.Referrer, event.UserAgent, event.IP,
		event.Country, event.City, event.Device, event.OS, event.Browser,
		propsJSON, event.Timestamp, event.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

func (r *ClickHouseRepository) BatchInsert(ctx context.Context, events []*domain.Event) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO events (
			id, type, user_id, session_id, tenant_id, url, referrer,
			user_agent, ip, country, city, device, os, browser,
			properties, timestamp, processed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		propsJSON := "{}"
		if len(event.Properties) > 0 {
			b, err := json.Marshal(event.Properties)
			if err != nil {
				return fmt.Errorf("failed to marshal properties: %w", err)
			}
			propsJSON = string(b)
		}

		if _, err := stmt.ExecContext(ctx,
			event.ID, event.Type, event.UserID, event.SessionID, event.TenantID,
			event.URL, event.Referrer, event.UserAgent, event.IP,
			event.Country, event.City, event.Device, event.OS, event.Browser,
			propsJSON, event.Timestamp, event.ProcessedAt,
		); err != nil {
			return fmt.Errorf("failed to execute batch insert: %w", err)
		}
	}

	return tx.Commit()
}

func (r *ClickHouseRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := `
		SELECT id, type, user_id, session_id, tenant_id, url, referrer,
			user_agent, ip, country, city, device, os, browser,
			properties, timestamp, processed_at
		FROM events WHERE id = $1 LIMIT 1
	`

	event := &domain.Event{}
	var propsJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID, &event.Type, &event.UserID, &event.SessionID, &event.TenantID,
		&event.URL, &event.Referrer, &event.UserAgent, &event.IP,
		&event.Country, &event.City, &event.Device, &event.OS, &event.Browser,
		&propsJSON, &event.Timestamp, &event.ProcessedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if propsJSON != "" && propsJSON != "{}" {
		if err := json.Unmarshal([]byte(propsJSON), &event.Properties); err != nil {
			return nil, fmt.Errorf("failed to unmarshal properties: %w", err)
		}
	}

	return event, nil
}

func (r *ClickHouseRepository) Query(ctx context.Context, filter domain.QueryFilter) ([]*domain.Event, error) {
	where, args, _ := buildFilterQueryCH(filter)

	limit := filter.Limit
	if limit <= 0 {
		limit = 1000
	}
	offset := filter.Offset

	query := fmt.Sprintf(`
		SELECT id, type, user_id, session_id, tenant_id, url, referrer,
			user_agent, ip, country, city, device, os, browser,
			properties, timestamp, processed_at
		FROM events %s
		ORDER BY timestamp DESC
		LIMIT %d OFFSET %d
	`, where, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		var propsJSON string

		if err := rows.Scan(
			&event.ID, &event.Type, &event.UserID, &event.SessionID, &event.TenantID,
			&event.URL, &event.Referrer, &event.UserAgent, &event.IP,
			&event.Country, &event.City, &event.Device, &event.OS, &event.Browser,
			&propsJSON, &event.Timestamp, &event.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if propsJSON != "" && propsJSON != "{}" {
			if err := json.Unmarshal([]byte(propsJSON), &event.Properties); err != nil {
				return nil, fmt.Errorf("failed to unmarshal properties: %w", err)
			}
		}

		events = append(events, event)
	}

	return events, nil
}

func (r *ClickHouseRepository) Count(ctx context.Context, filter domain.QueryFilter) (int64, error) {
	where, args, _ := buildFilterQueryCH(filter)

	query := fmt.Sprintf("SELECT COUNT(*) FROM events %s", where)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}

func (r *ClickHouseRepository) DeleteOld(ctx context.Context, tenantID string, retentionDays int) (int64, error) {
	query := `ALTER TABLE events DELETE WHERE tenant_id = $1 AND timestamp < $2`
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)

	result, err := r.db.ExecContext(ctx, query, tenantID, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected, nil
}

// MetricsRepository implementation for ClickHouse.

func (r *ClickHouseRepository) UpsertMetrics(ctx context.Context, metrics *domain.AggregatedMetrics) error {
	query := `
		INSERT INTO aggregated_metrics (
			id, tenant_id, window_start, window_end, granularity,
			total_events, unique_users, unique_sessions, page_views, clicks,
			conversions, avg_session_duration, bounce_rate, errors, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.ExecContext(ctx, query,
		metrics.ID, metrics.TenantID, metrics.WindowStart, metrics.WindowEnd, metrics.Granularity,
		metrics.TotalEvents, metrics.UniqueUsers, metrics.UniqueSessions, metrics.PageViews, metrics.Clicks,
		metrics.Conversions, metrics.AvgSessionDuration, metrics.BounceRate, metrics.Errors, metrics.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert metrics: %w", err)
	}

	return nil
}

func (r *ClickHouseRepository) BatchUpsertMetrics(ctx context.Context, metrics []*domain.AggregatedMetrics) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO aggregated_metrics (
			id, tenant_id, window_start, window_end, granularity,
			total_events, unique_users, unique_sessions, page_views, clicks,
			conversions, avg_session_duration, bounce_rate, errors, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, m := range metrics {
		if _, err := stmt.ExecContext(ctx,
			m.ID, m.TenantID, m.WindowStart, m.WindowEnd, m.Granularity,
			m.TotalEvents, m.UniqueUsers, m.UniqueSessions, m.PageViews, m.Clicks,
			m.Conversions, m.AvgSessionDuration, m.BounceRate, m.Errors, m.CreatedAt,
		); err != nil {
			return fmt.Errorf("failed to batch upsert metrics: %w", err)
		}
	}

	return tx.Commit()
}

func (r *ClickHouseRepository) GetTimeSeries(ctx context.Context, filter domain.QueryFilter) ([]domain.TimeSeriesPoint, error) {
	granularity := filter.Granularity
	if granularity == "" {
		granularity = "hour"
	}

	var truncateExpr string
	switch granularity {
	case "minute":
		truncateExpr = "toStartOfMinute(timestamp)"
	case "hour":
		truncateExpr = "toStartOfHour(timestamp)"
	case "day":
		truncateExpr = "toStartOfDay(timestamp)"
	default:
		truncateExpr = "toStartOfHour(timestamp)"
	}

	where, args, _ := buildFilterQueryCH(filter)

	query := fmt.Sprintf(`
		SELECT %s as ts, COUNT(*) as cnt
		FROM events %s
		GROUP BY ts
		ORDER BY ts
	`, truncateExpr, where)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}
	defer rows.Close()

	var points []domain.TimeSeriesPoint
	for rows.Next() {
		var ts time.Time
		var value float64
		if err := rows.Scan(&ts, &value); err != nil {
			return nil, fmt.Errorf("failed to scan time series point: %w", err)
		}
		points = append(points, domain.TimeSeriesPoint{
			Timestamp: ts,
			Value:     value,
		})
	}

	return points, nil
}

func (r *ClickHouseRepository) GetSummary(
	ctx context.Context,
	tenantID, metricName, from, to string,
) (*domain.MetricSummary, error) {
	query := `
		SELECT
			anyLast(value) as current,
			min(value) as min_val,
			max(value) as max_val,
			avg(value) as avg_val,
			quantile(0.50)(value) as p50,
			quantile(0.95)(value) as p95,
			quantile(0.99)(value) as p99
		FROM aggregated_metrics
		WHERE tenant_id = $1 AND window_start >= $2 AND window_end <= $3
	`

	summary := &domain.MetricSummary{
		Name: metricName,
	}

	var minVal, maxVal, avgVal, p50, p95, p99 float64
	err := r.db.QueryRowContext(ctx, query, tenantID, from, to).Scan(
		&summary.Current, &minVal, &maxVal, &avgVal, &p50, &p95, &p99,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get metric summary: %w", err)
	}

	summary.Min = minVal
	summary.Max = maxVal
	summary.Avg = avgVal
	summary.P50 = p50
	summary.P95 = p95
	summary.P99 = p99

	return summary, nil
}

func (r *ClickHouseRepository) GetByTenantAndWindow(
	ctx context.Context,
	tenantID, start, end, granularity string,
) ([]*domain.AggregatedMetrics, error) {
	query := `
		SELECT id, tenant_id, window_start, window_end, granularity,
			total_events, unique_users, unique_sessions, page_views, clicks,
			conversions, avg_session_duration, bounce_rate, errors, created_at
		FROM aggregated_metrics
		WHERE tenant_id = $1 AND window_start >= $2 AND window_end <= $3 AND granularity = $4
		ORDER BY window_start ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end, granularity)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by window: %w", err)
	}
	defer rows.Close()

	var metrics []*domain.AggregatedMetrics
	for rows.Next() {
		m := &domain.AggregatedMetrics{}
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.WindowStart, &m.WindowEnd, &m.Granularity,
			&m.TotalEvents, &m.UniqueUsers, &m.UniqueSessions, &m.PageViews, &m.Clicks,
			&m.Conversions, &m.AvgSessionDuration, &m.BounceRate, &m.Errors, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan metrics: %w", err)
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// serializeJSON and deserializeJSON helpers for the postgres file.
func serializeJSONCH(v interface{}) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func deserializeJSONCH(s string, v interface{}) error {
	if s == "" || s == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}

// buildFilterQuery constructs a dynamic SQL WHERE clause from QueryFilter.
func buildFilterQueryCH(filter domain.QueryFilter) (string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.TenantID != "" {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, filter.TenantID)
		argIdx++
	}

	if len(filter.EventTypes) > 0 {
		types := make([]string, len(filter.EventTypes))
		for i, t := range filter.EventTypes {
			types[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, string(t))
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(types, ",")))
	}

	if filter.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, filter.UserID)
		argIdx++
	}

	if filter.SessionID != "" {
		conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIdx))
		args = append(args, filter.SessionID)
		argIdx++
	}

	if filter.URL != "" {
		conditions = append(conditions, fmt.Sprintf("url = $%d", argIdx))
		args = append(args, filter.URL)
		argIdx++
	}

	if filter.Country != "" {
		conditions = append(conditions, fmt.Sprintf("country = $%d", argIdx))
		args = append(args, filter.Country)
		argIdx++
	}

	if !filter.DateFrom.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}

	if !filter.DateTo.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	return where, args, argIdx
}
