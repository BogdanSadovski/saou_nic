package domain

import (
	"time"

	"github.com/google/uuid"
)

// CollaboratorRole defines the role of an interviewer in the session
type CollaboratorRole string

const (
	RoleLead          CollaboratorRole = "lead"
	RoleCoInterviewer CollaboratorRole = "co-interviewer"
	RoleObserver      CollaboratorRole = "observer"
)

// InterviewCollaborator represents an interviewer participating in session
type InterviewCollaborator struct {
	ID        uuid.UUID        `json:"id"`
	SessionID uuid.UUID        `json:"session_id"`
	UserID    uuid.UUID        `json:"user_id"`
	Role      CollaboratorRole `json:"role"`
	JoinedAt  time.Time        `json:"joined_at"`
	LeftAt    *time.Time       `json:"left_at,omitempty"`
	IsActive  bool             `json:"is_active"`
	CreatedAt time.Time        `json:"created_at"`
}

// CollaborationNote represents a shared note during interview
type CollaborationNote struct {
	ID        uuid.UUID   `json:"id"`
	SessionID uuid.UUID   `json:"session_id"`
	AuthorID  uuid.UUID   `json:"author_id"`
	Content   string      `json:"content"`
	Version   int         `json:"version"`
	IsPinned  bool        `json:"is_pinned"`
	Mentions  []uuid.UUID `json:"mentions,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// InterviewerScore represents scoring by one interviewer
type InterviewerScore struct {
	ID            uuid.UUID `json:"id"`
	SessionID     uuid.UUID `json:"session_id"`
	InterviewerID uuid.UUID `json:"interviewer_id"`

	// Scores (0-10)
	TechnicalScore      *int `json:"technical_score,omitempty"`
	CommunicationScore  *int `json:"communication_score,omitempty"`
	ProblemSolvingScore *int `json:"problem_solving_score,omitempty"`
	CultureFitScore     *int `json:"culture_fit_score,omitempty"`
	CodingQualityScore  *int `json:"coding_quality_score,omitempty"`

	// Recommendation
	Recommendation *string `json:"recommendation,omitempty"` // STRONG_YES, YES, MAYBE, NO, STRONG_NO

	// Feedback
	Strengths           *string `json:"strengths,omitempty"`
	AreasForImprovement *string `json:"areas_for_improvement,omitempty"`
	AdditionalComments  *string `json:"additional_comments,omitempty"`

	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// InterviewConsensus represents aggregated scoring and recommendation
type InterviewConsensus struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`

	// Average scores
	AvgTechnicalScore      *float64 `json:"avg_technical_score,omitempty"`
	AvgCommunicationScore  *float64 `json:"avg_communication_score,omitempty"`
	AvgProblemSolvingScore *float64 `json:"avg_problem_solving_score,omitempty"`
	AvgCultureFitScore     *float64 `json:"avg_culture_fit_score,omitempty"`
	AvgCodingQualityScore  *float64 `json:"avg_coding_quality_score,omitempty"`

	// Disagreement metrics
	ScoreVariance     *float64 `json:"score_variance,omitempty"`
	DisagreementLevel *string  `json:"disagreement_level,omitempty"` // LOW, MEDIUM, HIGH

	// Consensus
	ConsensusRecommendation *string  `json:"consensus_recommendation,omitempty"`
	ConfidenceScore         *float64 `json:"confidence_score,omitempty"`

	Alignments map[string]interface{} `json:"alignments,omitempty"`

	CalculatedAt *time.Time `json:"calculated_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ScoreAuditLog tracks changes to scores
type ScoreAuditLog struct {
	ID            uuid.UUID              `json:"id"`
	SessionID     uuid.UUID              `json:"session_id"`
	InterviewerID uuid.UUID              `json:"interviewer_id"`
	Action        string                 `json:"action"` // CREATED, UPDATED, SUBMITTED, DELETED
	OldScores     map[string]interface{} `json:"old_scores,omitempty"`
	NewScores     map[string]interface{} `json:"new_scores,omitempty"`
	ChangeReason  *string                `json:"change_reason,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// AddCollaboratorRequest API request
type AddCollaboratorRequest struct {
	UserID uuid.UUID        `json:"user_id"`
	Role   CollaboratorRole `json:"role"`
}

// AddNoteRequest API request
type AddNoteRequest struct {
	Content  string      `json:"content"`
	IsPinned bool        `json:"is_pinned,omitempty"`
	Mentions []uuid.UUID `json:"mentions,omitempty"`
}

// SubmitScoreRequest API request
type SubmitScoreRequest struct {
	TechnicalScore      *int   `json:"technical_score"`
	CommunicationScore  *int   `json:"communication_score"`
	ProblemSolvingScore *int   `json:"problem_solving_score"`
	CultureFitScore     *int   `json:"culture_fit_score"`
	CodingQualityScore  *int   `json:"coding_quality_score"`
	Recommendation      string `json:"recommendation"`
	Strengths           string `json:"strengths"`
	AreasForImprovement string `json:"areas_for_improvement"`
	AdditionalComments  string `json:"additional_comments,omitempty"`
}
