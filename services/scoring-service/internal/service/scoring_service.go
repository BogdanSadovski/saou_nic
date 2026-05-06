package service

import (
	"context"
	"fmt"
	"time"

	"scoring-service/internal/domain"
	"scoring-service/pkg/scoring"

	"github.com/google/uuid"
)

// ScoringService orchestrates the scoring workflow.
type ScoringService struct {
	scoreRepo     domain.ScoreRepository
	rubricRepo    domain.RubricRepository
	evalEngine    *EvaluationEngine
	rubricCalc    *RubricCalculator
	passThreshold float64
}

// NewScoringService creates a new scoring service instance.
func NewScoringService(scoreRepo domain.ScoreRepository, rubricRepo domain.RubricRepository, passThreshold float64) *ScoringService {
	return &ScoringService{
		scoreRepo:     scoreRepo,
		rubricRepo:    rubricRepo,
		evalEngine:    NewEvaluationEngine(),
		rubricCalc:    NewRubricCalculator(),
		passThreshold: passThreshold,
	}
}

// Evaluate processes a scoring request and returns the computed score.
func (s *ScoringService) Evaluate(ctx context.Context, req domain.ScoringRequest) (*domain.Score, error) {
	// Initialize the score record
	score := &domain.Score{
		ID:           uuid.New().String(),
		SubmissionID: req.SubmissionID,
		ScoreType:    req.ScoreType,
		Status:       domain.ScoreStatusPending,
		RubricID:     req.RubricID,
		MaxScore:     100,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.scoreRepo.Create(ctx, score); err != nil {
		return nil, fmt.Errorf("create score record: %w", err)
	}

	// Collect metrics from the submission
	metrics, err := s.evalEngine.CollectMetrics(ctx, req)
	if err != nil {
		errMsg := err.Error()
		score.Status = domain.ScoreStatusFailed
		score.ErrorMessage = &errMsg
		score.UpdatedAt = time.Now()
		_ = s.scoreRepo.Update(ctx, score)
		return nil, fmt.Errorf("collect metrics: %w", err)
	}

	// Evaluate based on score type
	var breakdown []domain.CriterionScore

	if req.RubricID != nil {
		// Use rubric-based evaluation
		rubric, err := s.rubricRepo.GetByID(ctx, *req.RubricID)
		if err != nil {
			errMsg := fmt.Sprintf("rubric not found: %s", *req.RubricID)
			score.Status = domain.ScoreStatusFailed
			score.ErrorMessage = &errMsg
			score.UpdatedAt = time.Now()
			_ = s.scoreRepo.Update(ctx, score)
			return nil, fmt.Errorf("get rubric: %w", err)
		}

		breakdown = s.rubricCalc.CalculateFromRubric(rubric, *metrics)
	} else {
		// Use default algorithm evaluation
		results, err := s.evalEngine.Evaluate(ctx, req.ScoreType, *metrics)
		if err != nil {
			errMsg := err.Error()
			score.Status = domain.ScoreStatusFailed
			score.ErrorMessage = &errMsg
			score.UpdatedAt = time.Now()
			_ = s.scoreRepo.Update(ctx, score)
			return nil, fmt.Errorf("evaluate: %w", err)
		}

		breakdown = s.convertToCriterionScores(results)
	}

	// Calculate final scores
	totalScore := s.calculateTotalScore(breakdown)
	percentage := (totalScore / score.MaxScore) * 100
	grade := scoring.CalculateGrade(percentage)

	score.TotalScore = totalScore
	score.Percentage = percentage
	score.Grade = grade
	score.Breakdown = breakdown
	score.Status = domain.ScoreStatusCompleted
	score.UpdatedAt = time.Now()

	if err := s.scoreRepo.Update(ctx, score); err != nil {
		return nil, fmt.Errorf("update score: %w", err)
	}

	return score, nil
}

// GetScore retrieves a score by ID.
func (s *ScoringService) GetScore(ctx context.Context, id string) (*domain.Score, error) {
	return s.scoreRepo.GetByID(ctx, id)
}

// GetScoresBySubmission retrieves all scores for a submission.
func (s *ScoringService) GetScoresBySubmission(ctx context.Context, submissionID string) ([]domain.Score, error) {
	return s.scoreRepo.GetBySubmissionID(ctx, submissionID)
}

// GetEvaluationResult aggregates all scores for a submission into an evaluation result.
func (s *ScoringService) GetEvaluationResult(ctx context.Context, submissionID string) (*domain.EvaluationResult, error) {
	scores, err := s.scoreRepo.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("get scores: %w", err)
	}

	if len(scores) == 0 {
		return nil, fmt.Errorf("no scores found for submission: %s", submissionID)
	}

	result := &domain.EvaluationResult{
		Scores:      scores,
		EvaluatedAt: time.Now(),
	}

	// Calculate overall score using type weights
	typeScores := make(map[string]float64)
	for _, score := range scores {
		if score.Status == domain.ScoreStatusCompleted {
			typeScores[string(score.ScoreType)] = score.Percentage
		}
	}

	// Use configured weights for overall calculation
	weights := map[string]float64{
		"code_quality":  0.30,
		"performance":   0.20,
		"security":      0.25,
		"documentation": 0.10,
		"test_coverage": 0.15,
	}

	result.OverallScore = scoring.ApplyWeights(typeScores, weights)
	result.OverallGrade = scoring.CalculateGrade(result.OverallScore)
	result.PassThreshold = scoring.PassThreshold(result.OverallScore, s.passThreshold)

	return result, nil
}

// ListScores retrieves paginated scores.
func (s *ScoringService) ListScores(ctx context.Context, limit, offset int) ([]domain.Score, error) {
	return s.scoreRepo.List(ctx, limit, offset)
}

// DeleteScore removes a score record.
func (s *ScoringService) DeleteScore(ctx context.Context, id string) error {
	return s.scoreRepo.Delete(ctx, id)
}

// CreateRubric creates a new scoring rubric.
func (s *ScoringService) CreateRubric(ctx context.Context, rubric *domain.Rubric) error {
	rubric.ID = uuid.New().String()
	rubric.CreatedAt = time.Now()
	rubric.UpdatedAt = time.Now()
	return s.rubricRepo.Create(ctx, rubric)
}

// GetRubric retrieves a rubric by ID.
func (s *ScoringService) GetRubric(ctx context.Context, id string) (*domain.Rubric, error) {
	return s.rubricRepo.GetByID(ctx, id)
}

func (s *ScoringService) calculateTotalScore(breakdown []domain.CriterionScore) float64 {
	if len(breakdown) == 0 {
		return 0
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for _, cs := range breakdown {
		w := cs.Weight
		if w == 0 {
			w = 1.0
		}
		// cs.WeightedScore already has cs.Weight baked in by the producer; do
		// not multiply by w again here to avoid double-weighting. Just sum the
		// already-weighted scores and normalize by total weight.
		weightedSum += cs.WeightedScore
		totalWeight += w
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

func (s *ScoringService) convertToCriterionScores(results scoring.CriterionResults) []domain.CriterionScore {
	scores := make([]domain.CriterionScore, 0, len(results.Results))
	for _, r := range results.Results {
		weight := 1.0
		var weightedScore float64
		if r.MaxScore > 0 {
			weightedScore = (r.Score / r.MaxScore) * 100
		}

		scores = append(scores, domain.CriterionScore{
			CriterionName: r.Name,
			Score:         r.Score,
			MaxScore:      r.MaxScore,
			Weight:        weight,
			WeightedScore: weightedScore,
			Comments:      r.Comments,
		})
	}
	return scores
}
