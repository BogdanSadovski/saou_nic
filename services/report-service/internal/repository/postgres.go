package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bogdan/real_ass/report-service/internal/config"
	"github.com/bogdan/real_ass/report-service/internal/domain"

	_ "github.com/lib/pq"
)

// PostgresRepository implements domain.ReportRepository
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(cfg config.DatabaseConfig) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &PostgresRepository{db: db}, nil
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// Create inserts a new report record
func (r *PostgresRepository) Create(ctx context.Context, report *domain.Report) error {
	query := `
		INSERT INTO reports (
			id, candidate_id, interview_id, assessment_id, type, format, status,
			title, description, file_url, file_name, file_size, error_message,
			metadata, created_at, updated_at, expires_at, generated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.db.ExecContext(ctx, query,
		report.ID,
		report.CandidateID,
		report.InterviewID,
		report.AssessmentID,
		report.Type,
		report.Format,
		report.Status,
		report.Title,
		report.Description,
		report.FileURL,
		report.FileName,
		report.FileSize,
		report.ErrorMessage,
		report.Metadata,
		report.CreatedAt,
		report.UpdatedAt,
		report.ExpiresAt,
		report.GeneratedBy,
	)
	if err != nil {
		return fmt.Errorf("inserting report: %w", err)
	}
	return nil
}

// GetByID retrieves a report by its ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*domain.Report, error) {
	query := `
		SELECT id, candidate_id, interview_id, assessment_id, type, format, status,
			   title, description, file_url, file_name, file_size, error_message,
			   metadata, created_at, updated_at, expires_at, generated_by
		FROM reports WHERE id = $1
	`
	report := &domain.Report{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&report.ID,
		&report.CandidateID,
		&report.InterviewID,
		&report.AssessmentID,
		&report.Type,
		&report.Format,
		&report.Status,
		&report.Title,
		&report.Description,
		&report.FileURL,
		&report.FileName,
		&report.FileSize,
		&report.ErrorMessage,
		&report.Metadata,
		&report.CreatedAt,
		&report.UpdatedAt,
		&report.ExpiresAt,
		&report.GeneratedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found: %s", id)
		}
		return nil, fmt.Errorf("querying report: %w", err)
	}
	return report, nil
}

// List retrieves reports with pagination and optional filters
func (r *PostgresRepository) List(ctx context.Context, params domain.ListReportsParams) (*domain.ReportListResponse, error) {
	query := `
		SELECT id, candidate_id, interview_id, assessment_id, type, format, status,
			   title, description, file_url, file_name, file_size, error_message,
			   metadata, created_at, updated_at, expires_at, generated_by
		FROM reports WHERE 1=1
	`
	args := make([]interface{}, 0)
	argCount := 1

	if params.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *params.Status)
		argCount++
	}
	if params.Format != nil {
		query += fmt.Sprintf(" AND format = $%d", argCount)
		args = append(args, *params.Format)
		argCount++
	}
	if params.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *params.Type)
		argCount++
	}
	if params.CandidateID != "" {
		query += fmt.Sprintf(" AND candidate_id = $%d", argCount)
		args = append(args, params.CandidateID)
		argCount++
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", query)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("counting reports: %w", err)
	}

	// Add pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, params.PageSize, params.Page*params.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying reports: %w", err)
	}
	defer rows.Close()

	reports := make([]domain.Report, 0, params.PageSize)
	for rows.Next() {
		var report domain.Report
		if err := rows.Scan(
			&report.ID,
			&report.CandidateID,
			&report.InterviewID,
			&report.AssessmentID,
			&report.Type,
			&report.Format,
			&report.Status,
			&report.Title,
			&report.Description,
			&report.FileURL,
			&report.FileName,
			&report.FileSize,
			&report.ErrorMessage,
			&report.Metadata,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.ExpiresAt,
			&report.GeneratedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning report row: %w", err)
		}
		reports = append(reports, report)
	}

	return &domain.ReportListResponse{
		Reports:  reports,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
		HasMore:  int64(params.Page*params.PageSize+len(reports)) < total,
	}, nil
}

// Update updates an existing report record
func (r *PostgresRepository) Update(ctx context.Context, report *domain.Report) error {
	query := `
		UPDATE reports SET
			candidate_id = $2, interview_id = $3, assessment_id = $4, type = $5,
			format = $6, status = $7, title = $8, description = $9, file_url = $10,
			file_name = $11, file_size = $12, error_message = $13, metadata = $14,
			updated_at = $15, expires_at = $16, generated_by = $17
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		report.ID,
		report.CandidateID,
		report.InterviewID,
		report.AssessmentID,
		report.Type,
		report.Format,
		report.Status,
		report.Title,
		report.Description,
		report.FileURL,
		report.FileName,
		report.FileSize,
		report.ErrorMessage,
		report.Metadata,
		report.UpdatedAt,
		report.ExpiresAt,
		report.GeneratedBy,
	)
	if err != nil {
		return fmt.Errorf("updating report: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking update result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("report not found: %s", report.ID)
	}

	return nil
}

// UpdateStatus updates only the status and error message of a report
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, status domain.ReportStatus, errorMsg string) error {
	query := `UPDATE reports SET status = $2, error_message = $3, updated_at = $4 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status, errorMsg, time.Now())
	if err != nil {
		return fmt.Errorf("updating report status: %w", err)
	}
	return nil
}

// Delete removes a report record
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM reports WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting report: %w", err)
	}
	return nil
}

// DeleteExpired removes all reports past their expiration date
func (r *PostgresRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM reports WHERE expires_at < NOW() AND expires_at IS NOT NULL")
	if err != nil {
		return 0, fmt.Errorf("deleting expired reports: %w", err)
	}
	return result.RowsAffected()
}

// GetStats returns aggregated report statistics
func (r *PostgresRepository) GetStats(ctx context.Context) (*domain.ReportStats, error) {
	stats := &domain.ReportStats{
		ByStatus: make(map[string]int64),
		ByFormat: make(map[string]int64),
		ByType:   make(map[string]int64),
	}

	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reports").Scan(&stats.TotalReports); err != nil {
		return nil, fmt.Errorf("counting total reports: %w", err)
	}

	// Stats by status
	rows, err := r.db.QueryContext(ctx, "SELECT status, COUNT(*) FROM reports GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("querying status stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scanning status stats: %w", err)
		}
		stats.ByStatus[status] = count
	}

	// Stats by format
	rows, err = r.db.QueryContext(ctx, "SELECT format, COUNT(*) FROM reports GROUP BY format")
	if err != nil {
		return nil, fmt.Errorf("querying format stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var format string
		var count int64
		if err := rows.Scan(&format, &count); err != nil {
			return nil, fmt.Errorf("scanning format stats: %w", err)
		}
		stats.ByFormat[format] = count
	}

	// Stats by type
	rows, err = r.db.QueryContext(ctx, "SELECT type, COUNT(*) FROM reports GROUP BY type")
	if err != nil {
		return nil, fmt.Errorf("querying type stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reportType string
		var count int64
		if err := rows.Scan(&reportType, &count); err != nil {
			return nil, fmt.Errorf("scanning type stats: %w", err)
		}
		stats.ByType[reportType] = count
	}

	return stats, nil
}

// GetByCandidateID retrieves all reports for a specific candidate
func (r *PostgresRepository) GetByCandidateID(ctx context.Context, candidateID string) ([]domain.Report, error) {
	query := `
		SELECT id, candidate_id, interview_id, assessment_id, type, format, status,
			   title, description, file_url, file_name, file_size, error_message,
			   metadata, created_at, updated_at, expires_at, generated_by
		FROM reports WHERE candidate_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, candidateID)
	if err != nil {
		return nil, fmt.Errorf("querying reports by candidate: %w", err)
	}
	defer rows.Close()

	return scanReports(rows)
}

// GetPendingReports retrieves reports awaiting generation
func (r *PostgresRepository) GetPendingReports(ctx context.Context) ([]domain.Report, error) {
	query := `
		SELECT id, candidate_id, interview_id, assessment_id, type, format, status,
			   title, description, file_url, file_name, file_size, error_message,
			   metadata, created_at, updated_at, expires_at, generated_by
		FROM reports WHERE status = 'pending' ORDER BY created_at ASC LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying pending reports: %w", err)
	}
	defer rows.Close()

	return scanReports(rows)
}

// scanReports is a helper to scan multiple report rows
func scanReports(rows *sql.Rows) ([]domain.Report, error) {
	var reports []domain.Report
	for rows.Next() {
		var report domain.Report
		if err := rows.Scan(
			&report.ID,
			&report.CandidateID,
			&report.InterviewID,
			&report.AssessmentID,
			&report.Type,
			&report.Format,
			&report.Status,
			&report.Title,
			&report.Description,
			&report.FileURL,
			&report.FileName,
			&report.FileSize,
			&report.ErrorMessage,
			&report.Metadata,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.ExpiresAt,
			&report.GeneratedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning report row: %w", err)
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating report rows: %w", err)
	}
	return reports, nil
}
