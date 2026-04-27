package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"scoring-service/internal/domain"

	_ "github.com/lib/pq"
)

// PostgresRepository implements domain.Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository instance.
func NewPostgresRepository(dsn string, maxOpenConns, maxIdleConns int, connMaxLifetime string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	if connMaxLifetime != "" {
		duration, err := time.ParseDuration(connMaxLifetime)
		if err != nil {
			return nil, fmt.Errorf("parse conn max lifetime: %w", err)
		}
		db.SetConnMaxLifetime(duration)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &PostgresRepository{db: db}, nil
}

// Close closes the underlying database connection.
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// Score operations

func (r *PostgresRepository) Create(ctx context.Context, score *domain.Score) error {
	breakdownJSON, err := json.Marshal(score.Breakdown)
	if err != nil {
		return fmt.Errorf("marshal breakdown: %w", err)
	}

	query := `
		INSERT INTO scores (
			id, submission_id, score_type, total_score, max_score, percentage,
			grade, breakdown, status, rubric_id, error_message, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.db.ExecContext(ctx, query,
		score.ID,
		score.SubmissionID,
		score.ScoreType,
		score.TotalScore,
		score.MaxScore,
		score.Percentage,
		score.Grade,
		breakdownJSON,
		score.Status,
		score.RubricID,
		score.ErrorMessage,
		score.CreatedAt,
		score.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert score: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*domain.Score, error) {
	query := `
		SELECT id, submission_id, score_type, total_score, max_score, percentage,
			grade, breakdown, status, rubric_id, error_message, created_at, updated_at
		FROM scores WHERE id = $1
	`

	score := &domain.Score{}
	var breakdownJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&score.ID,
		&score.SubmissionID,
		&score.ScoreType,
		&score.TotalScore,
		&score.MaxScore,
		&score.Percentage,
		&score.Grade,
		&breakdownJSON,
		&score.Status,
		&score.RubricID,
		&score.ErrorMessage,
		&score.CreatedAt,
		&score.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("score not found: %s", id)
		}
		return nil, fmt.Errorf("query score: %w", err)
	}

	if err := json.Unmarshal(breakdownJSON, &score.Breakdown); err != nil {
		return nil, fmt.Errorf("unmarshal breakdown: %w", err)
	}

	return score, nil
}

func (r *PostgresRepository) GetBySubmissionID(ctx context.Context, submissionID string) ([]domain.Score, error) {
	query := `
		SELECT id, submission_id, score_type, total_score, max_score, percentage,
			grade, breakdown, status, rubric_id, error_message, created_at, updated_at
		FROM scores WHERE submission_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, submissionID)
	if err != nil {
		return nil, fmt.Errorf("query scores: %w", err)
	}
	defer rows.Close()

	var scores []domain.Score
	for rows.Next() {
		var score domain.Score
		var breakdownJSON []byte

		if err := rows.Scan(
			&score.ID,
			&score.SubmissionID,
			&score.ScoreType,
			&score.TotalScore,
			&score.MaxScore,
			&score.Percentage,
			&score.Grade,
			&breakdownJSON,
			&score.Status,
			&score.RubricID,
			&score.ErrorMessage,
			&score.CreatedAt,
			&score.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan score row: %w", err)
		}

		if err := json.Unmarshal(breakdownJSON, &score.Breakdown); err != nil {
			return nil, fmt.Errorf("unmarshal breakdown: %w", err)
		}

		scores = append(scores, score)
	}

	return scores, nil
}

func (r *PostgresRepository) Update(ctx context.Context, score *domain.Score) error {
	breakdownJSON, err := json.Marshal(score.Breakdown)
	if err != nil {
		return fmt.Errorf("marshal breakdown: %w", err)
	}

	query := `
		UPDATE scores SET
			total_score = $2, max_score = $3, percentage = $4, grade = $5,
			breakdown = $6, status = $7, rubric_id = $8, error_message = $9,
			updated_at = $10
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		score.ID,
		score.TotalScore,
		score.MaxScore,
		score.Percentage,
		score.Grade,
		breakdownJSON,
		score.Status,
		score.RubricID,
		score.ErrorMessage,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("score not found: %s", score.ID)
	}

	return nil
}

func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]domain.Score, error) {
	query := `
		SELECT id, submission_id, score_type, total_score, max_score, percentage,
			grade, breakdown, status, rubric_id, error_message, created_at, updated_at
		FROM scores ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query scores: %w", err)
	}
	defer rows.Close()

	var scores []domain.Score
	for rows.Next() {
		var score domain.Score
		var breakdownJSON []byte

		if err := rows.Scan(
			&score.ID,
			&score.SubmissionID,
			&score.ScoreType,
			&score.TotalScore,
			&score.MaxScore,
			&score.Percentage,
			&score.Grade,
			&breakdownJSON,
			&score.Status,
			&score.RubricID,
			&score.ErrorMessage,
			&score.CreatedAt,
			&score.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan score row: %w", err)
		}

		if err := json.Unmarshal(breakdownJSON, &score.Breakdown); err != nil {
			return nil, fmt.Errorf("unmarshal breakdown: %w", err)
		}

		scores = append(scores, score)
	}

	return scores, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM scores WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete score: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("score not found: %s", id)
	}

	return nil
}

// Rubric operations

func (r *PostgresRepository) CreateRubric(ctx context.Context, rubric *domain.Rubric) error {
	criteriaJSON, err := json.Marshal(rubric.Criteria)
	if err != nil {
		return fmt.Errorf("marshal criteria: %w", err)
	}

	query := `
		INSERT INTO rubrics (id, name, score_type, criteria, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.ExecContext(ctx, query,
		rubric.ID,
		rubric.Name,
		rubric.ScoreType,
		criteriaJSON,
		rubric.CreatedAt,
		rubric.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert rubric: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetRubricByID(ctx context.Context, id string) (*domain.Rubric, error) {
	query := `
		SELECT id, name, score_type, criteria, created_at, updated_at
		FROM rubrics WHERE id = $1
	`

	rubric := &domain.Rubric{}
	var criteriaJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rubric.ID,
		&rubric.Name,
		&rubric.ScoreType,
		&criteriaJSON,
		&rubric.CreatedAt,
		&rubric.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rubric not found: %s", id)
		}
		return nil, fmt.Errorf("query rubric: %w", err)
	}

	if err := json.Unmarshal(criteriaJSON, &rubric.Criteria); err != nil {
		return nil, fmt.Errorf("unmarshal criteria: %w", err)
	}

	return rubric, nil
}

func (r *PostgresRepository) GetByScoreType(ctx context.Context, scoreType domain.ScoreType) ([]domain.Rubric, error) {
	query := `
		SELECT id, name, score_type, criteria, created_at, updated_at
		FROM rubrics WHERE score_type = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, scoreType)
	if err != nil {
		return nil, fmt.Errorf("query rubrics: %w", err)
	}
	defer rows.Close()

	var rubrics []domain.Rubric
	for rows.Next() {
		var rubric domain.Rubric
		var criteriaJSON []byte

		if err := rows.Scan(
			&rubric.ID,
			&rubric.Name,
			&rubric.ScoreType,
			&criteriaJSON,
			&rubric.CreatedAt,
			&rubric.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan rubric row: %w", err)
		}

		if err := json.Unmarshal(criteriaJSON, &rubric.Criteria); err != nil {
			return nil, fmt.Errorf("unmarshal criteria: %w", err)
		}

		rubrics = append(rubrics, rubric)
	}

	return rubrics, nil
}

func (r *PostgresRepository) UpdateRubric(ctx context.Context, rubric *domain.Rubric) error {
	criteriaJSON, err := json.Marshal(rubric.Criteria)
	if err != nil {
		return fmt.Errorf("marshal criteria: %w", err)
	}

	query := `
		UPDATE rubrics SET name = $2, score_type = $3, criteria = $4, updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		rubric.ID,
		rubric.Name,
		rubric.ScoreType,
		criteriaJSON,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("update rubric: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rubric not found: %s", rubric.ID)
	}

	return nil
}

func (r *PostgresRepository) DeleteRubric(ctx context.Context, id string) error {
	query := `DELETE FROM rubrics WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete rubric: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rubric not found: %s", id)
	}

	return nil
}
