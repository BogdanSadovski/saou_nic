package service

import (
	"context"
	"fmt"

	"github.com/interview-platform/interview-service/internal/domain"

	"github.com/google/uuid"
)

//go:generate mockgen -source=interview_service.go -destination=mocks/mock_interview_service.go -package=mocks

type InterviewService struct {
	repo       *domain.Repository
	questionGen *QuestionGenerator
}

func NewInterviewService(repo *domain.Repository, questionGen *QuestionGenerator) *InterviewService {
	return &InterviewService{
		repo:       repo,
		questionGen: questionGen,
	}
}

func (s *InterviewService) CreateInterview(ctx context.Context, req *domain.CreateInterviewRequest) (*domain.Interview, error) {
	interview := &domain.Interview{
		ID:            uuid.New(),
		InterviewerID: req.InterviewerID,
		CandidateID:   req.CandidateID,
		Title:         req.Title,
		Description:   req.Description,
		Status:        domain.InterviewStatusScheduled,
		ScheduledAt:   req.ScheduledAt,
		Duration:      req.Duration,
		Language:      req.Language,
	}

	if err := s.repo.Interview.Create(ctx, interview); err != nil {
		return nil, fmt.Errorf("failed to create interview: %w", err)
	}

	// Generate questions for the interview
	questions, err := s.questionGen.GenerateQuestions(ctx, &QuestionGenerationRequest{
		InterviewID:  interview.ID,
		Difficulty:   req.Difficulty,
		Tags:         req.Tags,
		Count:        3,
		Duration:     req.Duration,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate questions: %w", err)
	}

	if err := s.repo.Question.CreateBatch(ctx, questions); err != nil {
		return nil, fmt.Errorf("failed to save questions: %w", err)
	}

	return interview, nil
}

func (s *InterviewService) GetInterview(ctx context.Context, id uuid.UUID) (*domain.InterviewResponse, error) {
	interview, err := s.repo.Interview.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get interview: %w", err)
	}

	response := &domain.InterviewResponse{
		Interview: *interview,
	}

	// Fetch questions (public view without solutions)
	questions, err := s.repo.Question.GetByInterviewID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get questions: %w", err)
	}

	for _, q := range questions {
		response.Questions = append(response.Questions, domain.PublicQuestion{
			ID:          q.ID,
			Title:       q.Title,
			Description: q.Description,
			Type:        q.Type,
			Difficulty:  q.Difficulty,
			Tags:        q.Tags,
			StarterCode: q.StarterCode,
			Points:      q.Points,
			Order:       q.Order,
		})
	}

	// Fetch session if exists
	session, err := s.repo.Session.GetByInterviewID(ctx, id)
	if err == nil && session != nil {
		response.Session = session
	}

	return response, nil
}

func (s *InterviewService) ListInterviews(ctx context.Context, interviewerID, candidateID *uuid.UUID, status *domain.InterviewStatus, limit, offset int) ([]*domain.Interview, error) {
	if interviewerID != nil {
		return s.repo.Interview.GetByInterviewerID(ctx, *interviewerID, limit, offset)
	}

	if candidateID != nil {
		return s.repo.Interview.GetByCandidateID(ctx, *candidateID, limit, offset)
	}

	return s.repo.Interview.List(ctx, status, limit, offset)
}

func (s *InterviewService) UpdateInterview(ctx context.Context, id uuid.UUID, req *domain.UpdateInterviewRequest) (*domain.Interview, error) {
	interview, err := s.repo.Interview.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get interview: %w", err)
	}

	if req.Title != "" {
		interview.Title = req.Title
	}
	if req.Description != "" {
		interview.Description = req.Description
	}
	if req.Status != "" {
		interview.Status = req.Status
	}
	if req.Duration > 0 {
		interview.Duration = req.Duration
	}
	if req.Language != "" {
		interview.Language = req.Language
	}

	if err := s.repo.Interview.Update(ctx, interview); err != nil {
		return nil, fmt.Errorf("failed to update interview: %w", err)
	}

	return interview, nil
}

func (s *InterviewService) CancelInterview(ctx context.Context, id uuid.UUID) error {
	interview, err := s.repo.Interview.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get interview: %w", err)
	}

	if interview.Status == domain.InterviewStatusCompleted {
		return fmt.Errorf("cannot cancel a completed interview")
	}

	interview.Status = domain.InterviewStatusCancelled

	return s.repo.Interview.Update(ctx, interview)
}

func (s *InterviewService) DeleteInterview(ctx context.Context, id uuid.UUID) error {
	// Verify interview exists
	_, err := s.repo.Interview.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get interview: %w", err)
	}

	// Delete will cascade to questions and sessions
	return s.repo.Interview.Delete(ctx, id)
}
