package domain

import (
	"context"
)

// ScoreRepository defines the interface for score persistence operations.
type ScoreRepository interface {
	Create(ctx context.Context, score *Score) error
	GetByID(ctx context.Context, id string) (*Score, error)
	GetBySubmissionID(ctx context.Context, submissionID string) ([]Score, error)
	Update(ctx context.Context, score *Score) error
	List(ctx context.Context, limit, offset int) ([]Score, error)
	Delete(ctx context.Context, id string) error
}

// RubricRepository defines the interface for rubric persistence operations.
type RubricRepository interface {
	Create(ctx context.Context, rubric *Rubric) error
	GetByID(ctx context.Context, id string) (*Rubric, error)
	GetByScoreType(ctx context.Context, scoreType ScoreType) ([]Rubric, error)
	Update(ctx context.Context, rubric *Rubric) error
	Delete(ctx context.Context, id string) error
}

// Repository aggregates all repository interfaces.
// Intentionally omitted because embedded interfaces have methods with identical names
// but different signatures, which causes an invalid combined interface.
