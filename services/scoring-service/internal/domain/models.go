package domain

import "time"

// ScoreType represents the category of a scoring evaluation.
type ScoreType string

const (
	ScoreTypeCodeQuality    ScoreType = "code_quality"
	ScoreTypePerformance    ScoreType = "performance"
	ScoreTypeSecurity       ScoreType = "security"
	ScoreTypeDocumentation  ScoreType = "documentation"
	ScoreTypeTestCoverage   ScoreType = "test_coverage"
)

// ScoreStatus represents the current state of a scoring record.
type ScoreStatus string

const (
	ScoreStatusPending   ScoreStatus = "pending"
	ScoreStatusCompleted ScoreStatus = "completed"
	ScoreStatusFailed    ScoreStatus = "failed"
)

// RubricCriterion defines a single criterion used in rubric-based evaluation.
type RubricCriterion struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MaxScore    float64 `json:"max_score"`
	Weight      float64 `json:"weight"`
}

// ScoringRequest represents an incoming request to evaluate a submission.
type ScoringRequest struct {
	SubmissionID  string                 `json:"submission_id"`
	RepositoryURL string                 `json:"repository_url"`
	CommitHash    string                 `json:"commit_hash"`
	ScoreType     ScoreType              `json:"score_type"`
	RubricID      *string                `json:"rubric_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Score represents the result of an evaluation.
type Score struct {
	ID            string             `json:"id"`
	SubmissionID  string             `json:"submission_id"`
	ScoreType     ScoreType          `json:"score_type"`
	TotalScore    float64            `json:"total_score"`
	MaxScore      float64            `json:"max_score"`
	Percentage    float64            `json:"percentage"`
	Grade         string             `json:"grade"`
	Breakdown     []CriterionScore   `json:"breakdown"`
	Status        ScoreStatus        `json:"status"`
	RubricID      *string            `json:"rubric_id,omitempty"`
	ErrorMessage  *string            `json:"error_message,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// CriterionScore holds the score for an individual rubric criterion.
type CriterionScore struct {
	CriterionName string   `json:"criterion_name"`
	Score         float64  `json:"score"`
	MaxScore      float64  `json:"max_score"`
	Weight        float64  `json:"weight"`
	WeightedScore float64  `json:"weighted_score"`
	Comments      []string `json:"comments,omitempty"`
}

// Rubric defines a scoring rubric with multiple criteria.
type Rubric struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	ScoreType  ScoreType          `json:"score_type"`
	Criteria   []RubricCriterion  `json:"criteria"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// EvaluationResult aggregates all scoring outputs for a submission.
type EvaluationResult struct {
	Scores       []Score  `json:"scores"`
	OverallScore float64  `json:"overall_score"`
	OverallGrade string   `json:"overall_grade"`
	PassThreshold bool    `json:"pass_threshold"`
	EvaluatedAt  time.Time `json:"evaluated_at"`
}
