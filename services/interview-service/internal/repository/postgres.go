package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/interview-platform/interview-service/internal/config"
	"github.com/interview-platform/interview-service/internal/domain"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresRepository struct {
	db     *sql.DB
	config *config.DatabaseConfig
	logger *logrus.Logger
}

func NewPostgresRepository(cfg *config.DatabaseConfig, logger *logrus.Logger) (*PostgresRepository, error) {
	dsn := cfg.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("successfully connected to postgresql")

	return &PostgresRepository{
		db:     db,
		config: cfg,
		logger: logger,
	}, nil
}

func (r *PostgresRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *PostgresRepository) DB() *sql.DB {
	return r.db
}

// scanInterview scans a single interview row from the database
func scanInterview(row *sql.Row) (*domain.Interview, error) {
	var i domain.Interview
	err := row.Scan(
		&i.ID,
		&i.InterviewerID,
		&i.CandidateID,
		&i.Title,
		&i.Description,
		&i.Status,
		&i.ScheduledAt,
		&i.Duration,
		&i.Language,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// scanInterviews scans multiple interview rows
func scanInterviews(rows *sql.Rows) ([]*domain.Interview, error) {
	defer rows.Close()

	var interviews []*domain.Interview
	for rows.Next() {
		var i domain.Interview
		if err := rows.Scan(
			&i.ID,
			&i.InterviewerID,
			&i.CandidateID,
			&i.Title,
			&i.Description,
			&i.Status,
			&i.ScheduledAt,
			&i.Duration,
			&i.Language,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		interviews = append(interviews, &i)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return interviews, nil
}

func generateID() uuid.UUID {
	return uuid.New()
}
