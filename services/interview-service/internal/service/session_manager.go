package service

import (
	"context"
	"fmt"
	"time"

	"github.com/interview-platform/interview-service/internal/domain"

	"github.com/google/uuid"
)

type SessionManager struct {
	repo *domain.Repository
}

func NewSessionManager(repo *domain.Repository) *SessionManager {
	return &SessionManager{
		repo: repo,
	}
}

func (sm *SessionManager) StartSession(ctx context.Context, interviewID uuid.UUID) (*domain.Session, error) {
	// Verify interview exists and is scheduled
	interview, err := sm.repo.Interview.GetByID(ctx, interviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get interview: %w", err)
	}

	if interview.Status != domain.InterviewStatusScheduled {
		return nil, fmt.Errorf("interview cannot be started, current status: %s", interview.Status)
	}

	// Check if session already exists
	existingSession, err := sm.repo.Session.GetByInterviewID(ctx, interviewID)
	if err == nil && existingSession != nil {
		return existingSession, nil
	}

	// Create new session
	session := &domain.Session{
		ID:            uuid.New(),
		InterviewID:   interviewID,
		Status:        domain.InterviewStatusInProgress,
		CurrentQIndex: 0,
		StartTime:     time.Now(),
		Score:         0,
	}

	if err := sm.repo.Session.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update interview status
	interview.Status = domain.InterviewStatusInProgress
	if err := sm.repo.Interview.Update(ctx, interview); err != nil {
		return nil, fmt.Errorf("failed to update interview status: %w", err)
	}

	return session, nil
}

func (sm *SessionManager) GetSession(ctx context.Context, sessionID uuid.UUID) (*domain.Session, error) {
	session, err := sm.repo.Session.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

func (sm *SessionManager) GetSessionByInterview(ctx context.Context, interviewID uuid.UUID) (*domain.Session, error) {
	session, err := sm.repo.Session.GetByInterviewID(ctx, interviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

func (sm *SessionManager) SubmitAnswer(ctx context.Context, sessionID, questionID uuid.UUID, code, language string) (*domain.Answer, error) {
	// Verify session exists and is in progress
	session, err := sm.repo.Session.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != domain.InterviewStatusInProgress {
		return nil, fmt.Errorf("session is not in progress")
	}

	// Get the question
	question, err := sm.repo.Question.GetByID(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get question: %w", err)
	}

	// Evaluate the answer (placeholder - would integrate with code execution service)
	isCorrect, score := evaluateAnswer(code, question)

	answer := &domain.Answer{
		ID:          uuid.New(),
		SessionID:   sessionID,
		QuestionID:  questionID,
		Code:        code,
		Language:    language,
		IsCorrect:   &isCorrect,
		Score:       score,
		SubmittedAt: time.Now(),
	}

	if err := sm.repo.Answer.Create(ctx, answer); err != nil {
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	// Update session score
	session.Score += score

	// Move to next question
	session.CurrentQIndex++

	// Check if all questions are answered
	questions, err := sm.repo.Question.GetByInterviewID(ctx, session.InterviewID)
	if err == nil && session.CurrentQIndex >= len(questions) {
		// Complete the session
		now := time.Now()
		session.EndTime = &now
		session.Status = domain.InterviewStatusCompleted

		// Update interview status
		interview, err := sm.repo.Interview.GetByID(ctx, session.InterviewID)
		if err == nil {
			interview.Status = domain.InterviewStatusCompleted
			sm.repo.Interview.Update(ctx, interview)
		}
	}

	if err := sm.repo.Session.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return answer, nil
}

func (sm *SessionManager) EndSession(ctx context.Context, sessionID uuid.UUID, feedback string) (*domain.Session, error) {
	session, err := sm.repo.Session.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != domain.InterviewStatusInProgress {
		return nil, fmt.Errorf("session is not in progress")
	}

	now := time.Now()
	session.EndTime = &now
	session.Status = domain.InterviewStatusCompleted
	session.Feedback = feedback

	if err := sm.repo.Session.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Update interview status
	interview, err := sm.repo.Interview.GetByID(ctx, session.InterviewID)
	if err == nil {
		interview.Status = domain.InterviewStatusCompleted
		sm.repo.Interview.Update(ctx, interview)
	}

	return session, nil
}

func (sm *SessionManager) GetSessionResults(ctx context.Context, sessionID uuid.UUID) (*SessionResults, error) {
	session, err := sm.repo.Session.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	answers, err := sm.repo.Answer.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get answers: %w", err)
	}

	questions, err := sm.repo.Question.GetByInterviewID(ctx, session.InterviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get questions: %w", err)
	}

	// Calculate statistics
	totalPossible := 0
	totalScore := 0
	correctCount := 0

	for _, q := range questions {
		totalPossible += q.Points

		for _, a := range answers {
			if a.QuestionID == q.ID {
				totalScore += a.Score
				if a.IsCorrect != nil && *a.IsCorrect {
					correctCount++
				}
			}
		}
	}

	var duration *time.Duration
	if session.EndTime != nil {
		d := session.EndTime.Sub(session.StartTime)
		duration = &d
	}

	return &SessionResults{
		Session:        session,
		Answers:        answers,
		TotalQuestions: len(questions),
		TotalPossible:  totalPossible,
		TotalScore:     totalScore,
		CorrectCount:   correctCount,
		Duration:       duration,
	}, nil
}

type SessionResults struct {
	Session        *domain.Session  `json:"session"`
	Answers        []*domain.Answer `json:"answers"`
	TotalQuestions int              `json:"total_questions"`
	TotalPossible  int              `json:"total_possible"`
	TotalScore     int              `json:"total_score"`
	CorrectCount   int              `json:"correct_count"`
	Duration       *time.Duration   `json:"duration,omitempty"`
}

// evaluateAnswer is a placeholder for actual code evaluation logic.
// In production, this would integrate with a code execution service.
func evaluateAnswer(code string, question *domain.Question) (bool, int) {
	// Placeholder: basic evaluation based on code length and question points
	if len(code) < 10 {
		return false, 0
	}

	// Simple heuristic: award partial credit based on code length
	// In reality, this would run test cases
	scorePercentage := 0.7 // Assume 70% correct for placeholder
	score := int(float64(question.Points) * scorePercentage)

	return scorePercentage >= 0.8, score
}
