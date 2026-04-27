package domain

import (
	"time"

	"github.com/google/uuid"
)

type InterviewStatus string

const (
	InterviewStatusScheduled  InterviewStatus = "scheduled"
	InterviewStatusInProgress InterviewStatus = "in_progress"
	InterviewStatusCompleted  InterviewStatus = "completed"
	InterviewStatusCancelled  InterviewStatus = "cancelled"
	InterviewStatusNoShow     InterviewStatus = "no_show"
)

type DifficultyLevel string

const (
	DifficultyEasy   DifficultyLevel = "easy"
	DifficultyMedium DifficultyLevel = "medium"
	DifficultyHard   DifficultyLevel = "hard"
	DifficultyExpert DifficultyLevel = "expert"
)

type QuestionType string

const (
	QuestionTypeCoding       QuestionType = "coding"
	QuestionTypeSystemDesign QuestionType = "system_design"
	QuestionTypeBehavioral   QuestionType = "behavioral"
	QuestionTypeDebugging    QuestionType = "debugging"
)

type Interview struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	InterviewerID uuid.UUID       `json:"interviewer_id" db:"interviewer_id"`
	CandidateID   uuid.UUID       `json:"candidate_id" db:"candidate_id"`
	Title         string          `json:"title" db:"title"`
	Description   string          `json:"description" db:"description"`
	Status        InterviewStatus `json:"status" db:"status"`
	ScheduledAt   time.Time       `json:"scheduled_at" db:"scheduled_at"`
	Duration      int             `json:"duration" db:"duration"` // in minutes
	Language      string          `json:"language" db:"language"` // programming language
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

type Question struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	InterviewID uuid.UUID       `json:"interview_id" db:"interview_id"`
	Title       string          `json:"title" db:"title"`
	Description string          `json:"description" db:"description"`
	Type        QuestionType    `json:"type" db:"type"`
	Difficulty  DifficultyLevel `json:"difficulty" db:"difficulty"`
	Tags        []string        `json:"tags" db:"tags"`
	StarterCode string          `json:"starter_code" db:"starter_code"`
	Solution    string          `json:"-" db:"solution"` // hidden from candidate
	TestCases   []TestCase      `json:"-" db:"test_cases"`
	Points      int             `json:"points" db:"points"`
	Order       int             `json:"order" db:"question_order"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
}

type TestCase struct {
	ID       uuid.UUID `json:"id"`
	Input    string    `json:"input"`
	Output   string    `json:"output"`
	IsHidden bool      `json:"is_hidden"`
}

type Session struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	InterviewID   uuid.UUID       `json:"interview_id" db:"interview_id"`
	Status        InterviewStatus `json:"status" db:"status"`
	CurrentQIndex int             `json:"current_question_index" db:"current_question_index"`
	StartTime     time.Time       `json:"start_time" db:"start_time"`
	EndTime       *time.Time      `json:"end_time,omitempty" db:"end_time"`
	Score         int             `json:"score" db:"score"`
	Feedback      string          `json:"feedback,omitempty" db:"feedback"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

type Answer struct {
	ID          uuid.UUID `json:"id" db:"id"`
	SessionID   uuid.UUID `json:"session_id" db:"session_id"`
	QuestionID  uuid.UUID `json:"question_id" db:"question_id"`
	Code        string    `json:"code" db:"code"`
	Language    string    `json:"language" db:"language"`
	IsCorrect   *bool     `json:"is_correct,omitempty" db:"is_correct"`
	Score       int       `json:"score" db:"score"`
	SubmittedAt time.Time `json:"submitted_at" db:"submitted_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type CreateInterviewRequest struct {
	InterviewerID uuid.UUID       `json:"interviewer_id" binding:"required"`
	CandidateID   uuid.UUID       `json:"candidate_id" binding:"required"`
	Title         string          `json:"title" binding:"required"`
	Description   string          `json:"description"`
	ScheduledAt   time.Time       `json:"scheduled_at" binding:"required"`
	Duration      int             `json:"duration" binding:"required,min=15,max=180"`
	Language      string          `json:"language"`
	Difficulty    DifficultyLevel `json:"difficulty"`
	Tags          []string        `json:"tags"`
}

type UpdateInterviewRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      InterviewStatus `json:"status"`
	Duration    int             `json:"duration"`
	Language    string          `json:"language"`
}

type InterviewResponse struct {
	Interview
	Questions []PublicQuestion `json:"questions,omitempty"`
	Session   *Session         `json:"session,omitempty"`
}

type PublicQuestion struct {
	ID          uuid.UUID       `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Type        QuestionType    `json:"type"`
	Difficulty  DifficultyLevel `json:"difficulty"`
	Tags        []string        `json:"tags"`
	StarterCode string          `json:"starter_code"`
	Points      int             `json:"points"`
	Order       int             `json:"order"`
}

type InterviewSessionLevel string

const (
	InterviewSessionLevelJunior InterviewSessionLevel = "junior"
	InterviewSessionLevelMiddle InterviewSessionLevel = "middle"
	InterviewSessionLevelSenior InterviewSessionLevel = "senior"
)

type InterviewSessionStatus string

const (
	InterviewSessionStatusCreated  InterviewSessionStatus = "created"
	InterviewSessionStatusActive   InterviewSessionStatus = "active"
	InterviewSessionStatusFinished InterviewSessionStatus = "finished"
	InterviewSessionStatusFailed   InterviewSessionStatus = "failed"
)

type MessageSender string

const (
	MessageSenderAI     MessageSender = "ai"
	MessageSenderUser   MessageSender = "user"
	MessageSenderSystem MessageSender = "system"
)

type InterviewModuleSession struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	UserID          uuid.UUID              `json:"user_id" db:"user_id"`
	Role            string                 `json:"role" db:"role"`
	Level           InterviewSessionLevel  `json:"level" db:"level"`
	Status          InterviewSessionStatus `json:"status" db:"status"`
	CurrentTopic    string                 `json:"current_topic" db:"current_topic"`
	DifficultyScore int                    `json:"difficulty_score" db:"difficulty_score"`
	PressureLevel   int                    `json:"pressure_level" db:"pressure_level"`
	QuestionCount   int                    `json:"question_count" db:"question_count"`
	QuestionLimit   int                    `json:"question_limit" db:"question_limit"`
	StartedAt       time.Time              `json:"started_at" db:"started_at"`
	EndedAt         *time.Time             `json:"ended_at,omitempty" db:"ended_at"`
	DurationSeconds int                    `json:"duration_seconds" db:"duration_seconds"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

type InterviewModuleMessage struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	SessionID  uuid.UUID              `json:"session_id" db:"session_id"`
	Sender     MessageSender          `json:"sender" db:"sender"`
	Content    string                 `json:"content" db:"content"`
	Topic      *string                `json:"topic,omitempty" db:"topic"`
	Difficulty *int                   `json:"difficulty,omitempty" db:"difficulty"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	TokenUsage map[string]interface{} `json:"token_usage,omitempty" db:"token_usage"`
}

type InterviewModuleReport struct {
	ID              uuid.UUID                `json:"id" db:"id"`
	SessionID       uuid.UUID                `json:"session_id" db:"session_id"`
	Correctness     float64                  `json:"correctness" db:"correctness"`
	Clarity         float64                  `json:"clarity" db:"clarity"`
	Completeness    float64                  `json:"completeness" db:"completeness"`
	Relevance       float64                  `json:"relevance" db:"relevance"`
	OverallScore    float64                  `json:"overall_score" db:"overall_score"`
	Strengths       []map[string]interface{} `json:"strengths" db:"strengths"`
	Weaknesses      []map[string]interface{} `json:"weaknesses" db:"weaknesses"`
	Recommendations []map[string]interface{} `json:"recommendations" db:"recommendations"`
	GeneratedAt     time.Time                `json:"generated_at" db:"generated_at"`
}

type RequestLog struct {
	IdempotencyKey string    `json:"idempotency_key" db:"idempotency_key"`
	SessionID      uuid.UUID `json:"session_id" db:"session_id"`
	ResponseHash   string    `json:"response_hash" db:"response_hash"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
