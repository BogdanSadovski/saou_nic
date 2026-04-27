package repository

import (
	"context"

	"scoring-service/internal/domain"
)

type ScoreRepositoryAdapter struct {
	repo *PostgresRepository
}

func NewScoreRepositoryAdapter(repo *PostgresRepository) *ScoreRepositoryAdapter {
	return &ScoreRepositoryAdapter{repo: repo}
}

func (a *ScoreRepositoryAdapter) Create(ctx context.Context, score *domain.Score) error {
	return a.repo.Create(ctx, score)
}

func (a *ScoreRepositoryAdapter) GetByID(ctx context.Context, id string) (*domain.Score, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *ScoreRepositoryAdapter) GetBySubmissionID(ctx context.Context, submissionID string) ([]domain.Score, error) {
	return a.repo.GetBySubmissionID(ctx, submissionID)
}

func (a *ScoreRepositoryAdapter) Update(ctx context.Context, score *domain.Score) error {
	return a.repo.Update(ctx, score)
}

func (a *ScoreRepositoryAdapter) List(ctx context.Context, limit, offset int) ([]domain.Score, error) {
	return a.repo.List(ctx, limit, offset)
}

func (a *ScoreRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

type RubricRepositoryAdapter struct {
	repo *PostgresRepository
}

func NewRubricRepositoryAdapter(repo *PostgresRepository) *RubricRepositoryAdapter {
	return &RubricRepositoryAdapter{repo: repo}
}

func (a *RubricRepositoryAdapter) Create(ctx context.Context, rubric *domain.Rubric) error {
	return a.repo.CreateRubric(ctx, rubric)
}

func (a *RubricRepositoryAdapter) GetByID(ctx context.Context, id string) (*domain.Rubric, error) {
	return a.repo.GetRubricByID(ctx, id)
}

func (a *RubricRepositoryAdapter) GetByScoreType(ctx context.Context, scoreType domain.ScoreType) ([]domain.Rubric, error) {
	return a.repo.GetByScoreType(ctx, scoreType)
}

func (a *RubricRepositoryAdapter) Update(ctx context.Context, rubric *domain.Rubric) error {
	return a.repo.UpdateRubric(ctx, rubric)
}

func (a *RubricRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.DeleteRubric(ctx, id)
}
