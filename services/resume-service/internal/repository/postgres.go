package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"resume-service/internal/config"
	"resume-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements domain.ResumeRepository
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(cfg config.DatabaseConfig) (*PostgresRepository, error) {
	dsn := cfg.DSN()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

// Close closes the database connection pool
func (r *PostgresRepository) Close() {
	r.pool.Close()
}

// Ping checks database connectivity
func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Create inserts a new resume record
func (r *PostgresRepository) Create(ctx context.Context, resume *domain.Resume) error {
	query := `
		INSERT INTO resumes (
			id, user_id, file_name, file_url, content_type, status,
			first_name, last_name, email, phone, summary,
			skills, experience, education, languages, certifications,
			metadata, created_at, updated_at, error
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20
		)
	`

	skillsJSON, _ := json.Marshal(resume.Skills)
	experienceJSON, _ := json.Marshal(resume.Experience)
	educationJSON, _ := json.Marshal(resume.Education)
	languagesJSON, _ := json.Marshal(resume.Languages)
	certificationsJSON, _ := json.Marshal(resume.Certifications)
	metadataJSON, _ := json.Marshal(resume.Metadata)

	now := time.Now()
	resume.CreatedAt = now
	resume.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		resume.ID, resume.UserID, resume.FileName, resume.FileURL, resume.ContentType, resume.Status,
		resume.FirstName, resume.LastName, resume.Email, resume.Phone, resume.Summary,
		skillsJSON, experienceJSON, educationJSON, languagesJSON, certificationsJSON,
		metadataJSON, resume.CreatedAt, resume.UpdatedAt, resume.Error,
	)

	if err != nil {
		return fmt.Errorf("failed to create resume: %w", err)
	}

	return nil
}

// GetByID retrieves a resume by its ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*domain.Resume, error) {
	query := `
		SELECT id, user_id, file_name, file_url, content_type, status,
			   first_name, last_name, email, phone, summary,
			   skills, experience, education, languages, certifications,
			   metadata, created_at, updated_at, error
		FROM resumes
		WHERE id = $1
	`

	resume := &domain.Resume{}
	var skillsJSON, experienceJSON, educationJSON, languagesJSON, certificationsJSON, metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&resume.ID, &resume.UserID, &resume.FileName, &resume.FileURL, &resume.ContentType, &resume.Status,
		&resume.FirstName, &resume.LastName, &resume.Email, &resume.Phone, &resume.Summary,
		&skillsJSON, &experienceJSON, &educationJSON, &languagesJSON, &certificationsJSON,
		&metadataJSON, &resume.CreatedAt, &resume.UpdatedAt, &resume.Error,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("resume not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get resume: %w", err)
	}

	_ = json.Unmarshal(skillsJSON, &resume.Skills)
	_ = json.Unmarshal(experienceJSON, &resume.Experience)
	_ = json.Unmarshal(educationJSON, &resume.Education)
	_ = json.Unmarshal(languagesJSON, &resume.Languages)
	_ = json.Unmarshal(certificationsJSON, &resume.Certifications)
	_ = json.Unmarshal(metadataJSON, &resume.Metadata)

	return resume, nil
}

// GetByUserID retrieves all resumes for a specific user
func (r *PostgresRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Resume, error) {
	query := `
		SELECT id, user_id, file_name, file_url, content_type, status,
			   first_name, last_name, email, phone, summary,
			   skills, experience, education, languages, certifications,
			   metadata, created_at, updated_at, error
		FROM resumes
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query resumes: %w", err)
	}
	defer rows.Close()

	return r.scanResumes(rows)
}

// Update modifies an existing resume
func (r *PostgresRepository) Update(ctx context.Context, resume *domain.Resume) error {
	query := `
		UPDATE resumes SET
			file_name = $1, file_url = $2, content_type = $3, status = $4,
			first_name = $5, last_name = $6, email = $7, phone = $8, summary = $9,
			skills = $10, experience = $11, education = $12, languages = $13,
			certifications = $14, metadata = $15, updated_at = $16, error = $17
		WHERE id = $18
	`

	skillsJSON, _ := json.Marshal(resume.Skills)
	experienceJSON, _ := json.Marshal(resume.Experience)
	educationJSON, _ := json.Marshal(resume.Education)
	languagesJSON, _ := json.Marshal(resume.Languages)
	certificationsJSON, _ := json.Marshal(resume.Certifications)
	metadataJSON, _ := json.Marshal(resume.Metadata)

	resume.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		resume.FileName, resume.FileURL, resume.ContentType, resume.Status,
		resume.FirstName, resume.LastName, resume.Email, resume.Phone, resume.Summary,
		skillsJSON, experienceJSON, educationJSON, languagesJSON, certificationsJSON,
		metadataJSON, resume.UpdatedAt, resume.Error, resume.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update resume: %w", err)
	}

	return nil
}

// Delete removes a resume by its ID
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM resumes WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete resume: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("resume not found: %s", id)
	}

	return nil
}

// List retrieves resumes with optional filtering
func (r *PostgresRepository) List(ctx context.Context, filter *domain.ResumeFilter) ([]*domain.Resume, int, error) {
	query := `
		SELECT id, user_id, file_name, file_url, content_type, status,
			   first_name, last_name, email, phone, summary,
			   skills, experience, education, languages, certifications,
			   metadata, created_at, updated_at, error
		FROM resumes
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM resumes WHERE 1=1`

	args := make([]interface{}, 0)
	argIndex := 1

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query resumes: %w", err)
	}
	defer rows.Close()

	resumes, err := r.scanResumes(rows)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	var total int
	err = r.pool.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count resumes: %w", err)
	}

	return resumes, total, nil
}

// UpdateStatus updates only the status and error fields
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, status domain.ResumeStatus, errMsg string) error {
	query := `UPDATE resumes SET status = $1, error = $2, updated_at = $3 WHERE id = $4`

	_, err := r.pool.Exec(ctx, query, status, errMsg, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update resume status: %w", err)
	}

	return nil
}

// scanResumes is a helper to scan rows into Resume slices
func (r *PostgresRepository) scanResumes(rows pgx.Rows) ([]*domain.Resume, error) {
	var resumes []*domain.Resume

	for rows.Next() {
		resume := &domain.Resume{}
		var skillsJSON, experienceJSON, educationJSON, languagesJSON, certificationsJSON, metadataJSON []byte

		err := rows.Scan(
			&resume.ID, &resume.UserID, &resume.FileName, &resume.FileURL, &resume.ContentType, &resume.Status,
			&resume.FirstName, &resume.LastName, &resume.Email, &resume.Phone, &resume.Summary,
			&skillsJSON, &experienceJSON, &educationJSON, &languagesJSON, &certificationsJSON,
			&metadataJSON, &resume.CreatedAt, &resume.UpdatedAt, &resume.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resume: %w", err)
		}

		_ = json.Unmarshal(skillsJSON, &resume.Skills)
		_ = json.Unmarshal(experienceJSON, &resume.Experience)
		_ = json.Unmarshal(educationJSON, &resume.Education)
		_ = json.Unmarshal(languagesJSON, &resume.Languages)
		_ = json.Unmarshal(certificationsJSON, &resume.Certifications)
		_ = json.Unmarshal(metadataJSON, &resume.Metadata)

		resumes = append(resumes, resume)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return resumes, nil
}
