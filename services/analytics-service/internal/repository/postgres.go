package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"analytics-service/internal/config"
	"analytics-service/internal/domain"

	_ "github.com/lib/pq"
)

// PostgresRepository implements domain repositories using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(cfg config.PostgresConfig) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MinConns)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresRepository{db: db}, nil
}

// Close closes the database connection.
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// DashboardRepository implementation.

func (r *PostgresRepository) CreateDashboard(ctx context.Context, dashboard *domain.Dashboard) error {
	query := `
		INSERT INTO dashboards (id, name, tenant_id, description, widgets, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	widgetsJSON, err := serializeJSON(dashboard.Widgets)
	if err != nil {
		return fmt.Errorf("failed to serialize widgets: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		dashboard.ID, dashboard.Name, dashboard.TenantID, dashboard.Description,
		widgetsJSON, dashboard.CreatedBy, dashboard.CreatedAt, dashboard.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert dashboard: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetDashboard(ctx context.Context, id string) (*domain.Dashboard, error) {
	query := `SELECT id, name, tenant_id, description, widgets, created_by, created_at, updated_at FROM dashboards WHERE id = $1`

	d := &domain.Dashboard{}
	var widgetsJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.Name, &d.TenantID, &d.Description, &widgetsJSON,
		&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dashboard not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	if err := deserializeJSON(widgetsJSON, &d.Widgets); err != nil {
		return nil, fmt.Errorf("failed to deserialize widgets: %w", err)
	}

	return d, nil
}

func (r *PostgresRepository) ListDashboards(ctx context.Context, tenantID string) ([]*domain.Dashboard, error) {
	query := `SELECT id, name, tenant_id, description, widgets, created_by, created_at, updated_at FROM dashboards WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dashboards: %w", err)
	}
	defer rows.Close()

	var dashboards []*domain.Dashboard
	for rows.Next() {
		d := &domain.Dashboard{}
		var widgetsJSON string

		if err := rows.Scan(&d.ID, &d.Name, &d.TenantID, &d.Description, &widgetsJSON, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan dashboard: %w", err)
		}

		if err := deserializeJSON(widgetsJSON, &d.Widgets); err != nil {
			return nil, fmt.Errorf("failed to deserialize widgets: %w", err)
		}

		dashboards = append(dashboards, d)
	}

	return dashboards, nil
}

func (r *PostgresRepository) UpdateDashboard(ctx context.Context, dashboard *domain.Dashboard) error {
	query := `UPDATE dashboards SET name = $2, description = $3, widgets = $4, updated_at = $5 WHERE id = $1`

	widgetsJSON, err := serializeJSON(dashboard.Widgets)
	if err != nil {
		return fmt.Errorf("failed to serialize widgets: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		dashboard.ID, dashboard.Name, dashboard.Description, widgetsJSON, dashboard.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	return nil
}

func (r *PostgresRepository) DeleteDashboard(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM dashboards WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	return nil
}

// ExportRepository implementation.

func (r *PostgresRepository) CreateExport(ctx context.Context, req *domain.ExportRequest) error {
	query := `
		INSERT INTO export_requests (id, tenant_id, format, filter, status, file_url, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	filterJSON, err := serializeJSON(req.Filter)
	if err != nil {
		return fmt.Errorf("failed to serialize filter: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		req.ID, req.TenantID, req.Format, filterJSON, req.Status,
		req.FileURL, req.Error, req.CreatedAt, req.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert export request: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetExport(ctx context.Context, id string) (*domain.ExportRequest, error) {
	query := `SELECT id, tenant_id, format, filter, status, file_url, error, created_at, updated_at FROM export_requests WHERE id = $1`

	req := &domain.ExportRequest{}
	var filterJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.TenantID, &req.Format, &filterJSON, &req.Status,
		&req.FileURL, &req.Error, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("export request not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get export request: %w", err)
	}

	if err := deserializeJSON(filterJSON, &req.Filter); err != nil {
		return nil, fmt.Errorf("failed to deserialize filter: %w", err)
	}

	return req, nil
}

func (r *PostgresRepository) ListExports(ctx context.Context, tenantID string, limit, offset int) ([]*domain.ExportRequest, error) {
	query := `
		SELECT id, tenant_id, format, filter, status, file_url, error, created_at, updated_at
		FROM export_requests
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query exports: %w", err)
	}
	defer rows.Close()

	var exports []*domain.ExportRequest
	for rows.Next() {
		req := &domain.ExportRequest{}
		var filterJSON string

		if err := rows.Scan(&req.ID, &req.TenantID, &req.Format, &filterJSON, &req.Status, &req.FileURL, &req.Error, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan export: %w", err)
		}

		if err := deserializeJSON(filterJSON, &req.Filter); err != nil {
			return nil, fmt.Errorf("failed to deserialize filter: %w", err)
		}

		exports = append(exports, req)
	}

	return exports, nil
}

func (r *PostgresRepository) UpdateExport(ctx context.Context, req *domain.ExportRequest) error {
	query := `
		UPDATE export_requests SET status = $2, file_url = $3, error = $4, updated_at = $5 WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, req.ID, req.Status, req.FileURL, req.Error, req.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update export: %w", err)
	}

	return nil
}

// UserRepository implementation.

func (r *PostgresRepository) TrackSession(ctx context.Context, userID, sessionID string, duration float64) error {
	query := `
		INSERT INTO user_sessions (user_id, session_id, duration, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id) DO UPDATE SET duration = $3
	`
	_, err := r.db.ExecContext(ctx, query, userID, sessionID, duration, time.Now().UTC())
	return err
}

func (r *PostgresRepository) GetActiveUsers(ctx context.Context, tenantID, from, to string) (int64, error) {
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM user_sessions
		WHERE created_at >= $1 AND created_at <= $2
	`
	var count int64
	err := r.db.QueryRowContext(ctx, query, from, to).Scan(&count)
	return count, err
}

func (r *PostgresRepository) GetUserSegments(ctx context.Context, tenantID, from, to string) (map[string]int64, error) {
	query := `
		SELECT country, COUNT(DISTINCT user_id) as cnt
		FROM events
		WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3 AND country != ''
		GROUP BY country
		ORDER BY cnt DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := make(map[string]int64)
	for rows.Next() {
		var country string
		var count int64
		if err := rows.Scan(&country, &count); err != nil {
			return nil, err
		}
		segments[country] = count
	}

	return segments, nil
}

func (r *PostgresRepository) GetTopUsers(ctx context.Context, tenantID, from, to string, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT user_id, COUNT(*) as event_count, COUNT(DISTINCT session_id) as session_count
		FROM events
		WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3 AND user_id != ''
		GROUP BY user_id
		ORDER BY event_count DESC
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var userID string
		var eventCount, sessionCount int64
		if err := rows.Scan(&userID, &eventCount, &sessionCount); err != nil {
			return nil, err
		}
		users = append(users, map[string]interface{}{
			"user_id":       userID,
			"event_count":   eventCount,
			"session_count": sessionCount,
		})
	}

	return users, nil
}

// JSON serialization helpers.
func serializeJSON(v interface{}) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func deserializeJSON(s string, v interface{}) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}

// buildFilterQuery constructs a dynamic SQL WHERE clause from QueryFilter.
func buildFilterQuery(filter domain.QueryFilter) (string, []interface{}, int) {
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

// ==================== FunnelRepository Implementation ====================

func (r *PostgresRepository) CreateFunnel(ctx context.Context, funnel *domain.Funnel) error {
	query := `
		INSERT INTO funnels (id, name, tenant_id, steps, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	stepsJSON, err := json.Marshal(funnel.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal funnel steps: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		funnel.ID, funnel.Name, funnel.TenantID, string(stepsJSON),
		funnel.CreatedAt, funnel.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create funnel: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetFunnel(ctx context.Context, id string) (*domain.Funnel, error) {
	query := `SELECT id, name, tenant_id, steps, created_at, updated_at FROM funnels WHERE id = $1`

	funnel := &domain.Funnel{}
	var stepsJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&funnel.ID, &funnel.Name, &funnel.TenantID, &stepsJSON,
		&funnel.CreatedAt, &funnel.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("funnel not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get funnel: %w", err)
	}

	if err := json.Unmarshal([]byte(stepsJSON), &funnel.Steps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal funnel steps: %w", err)
	}

	return funnel, nil
}

func (r *PostgresRepository) ListFunnels(ctx context.Context, tenantID string) ([]*domain.Funnel, error) {
	query := `SELECT id, name, tenant_id, steps, created_at, updated_at FROM funnels WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query funnels: %w", err)
	}
	defer rows.Close()

	var funnels []*domain.Funnel
	for rows.Next() {
		f := &domain.Funnel{}
		var stepsJSON string

		if err := rows.Scan(&f.ID, &f.Name, &f.TenantID, &stepsJSON, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan funnel: %w", err)
		}

		if err := json.Unmarshal([]byte(stepsJSON), &f.Steps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal funnel steps: %w", err)
		}

		funnels = append(funnels, f)
	}

	return funnels, nil
}

func (r *PostgresRepository) UpdateFunnel(ctx context.Context, funnel *domain.Funnel) error {
	query := `UPDATE funnels SET name = $2, steps = $3, updated_at = $4 WHERE id = $1`

	stepsJSON, err := json.Marshal(funnel.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal funnel steps: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, funnel.ID, funnel.Name, string(stepsJSON), funnel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update funnel: %w", err)
	}

	return nil
}

func (r *PostgresRepository) DeleteFunnel(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM funnels WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete funnel: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ComputeFunnel(ctx context.Context, funnelID string, from, to string) (*domain.FunnelResult, error) {
	funnel, err := r.GetFunnel(ctx, funnelID)
	if err != nil {
		return nil, err
	}

	result := &domain.FunnelResult{
		FunnelID:   funnelID,
		ComputedAt: time.Now().UTC(),
	}

	// Query events for the first step to get total starters.
	if len(funnel.Steps) == 0 {
		return result, nil
	}

	firstStep := funnel.Steps[0]
	types := make([]string, len(firstStep.EventTypes))
	for i, t := range firstStep.EventTypes {
		types[i] = string(t)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT user_id) FROM events
		WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3
		AND type IN (%s)
	`, strings.Join(types, ","))

	var totalStart int64
	err = r.db.QueryRowContext(ctx, query, funnel.TenantID, from, to).Scan(&totalStart)
	if err != nil {
		return nil, fmt.Errorf("failed to compute funnel start count: %w", err)
	}

	result.TotalStart = totalStart

	// Compute each step.
	for i, step := range funnel.Steps {
		stepTypes := make([]string, len(step.EventTypes))
		for j, t := range step.EventTypes {
			stepTypes[j] = string(t)
		}

		stepQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT user_id) FROM events
			WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3
			AND type IN (%s)
		`, strings.Join(stepTypes, ","))

		var count int64
		err = r.db.QueryRowContext(ctx, stepQuery, funnel.TenantID, from, to).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to compute funnel step %d: %w", i, err)
		}

		stepResult := domain.FunnelStepResult{
			StepNumber: step.StepNumber,
			Name:       step.Name,
			Count:      count,
		}

		if totalStart > 0 {
			stepResult.ConversionRate = float64(count) / float64(totalStart) * 100
		}
		if i > 0 && result.Steps[i-1].Count > 0 {
			stepResult.DropOffRate = (1 - float64(count)/float64(result.Steps[i-1].Count)) * 100
		} else if i == 0 {
			stepResult.DropOffRate = 0
		}

		result.Steps = append(result.Steps, stepResult)
	}

	return result, nil
}
