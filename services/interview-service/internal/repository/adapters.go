package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/interview-platform/interview-service/internal/domain"
)

// InterviewRepoAdapter adapts PostgresRepository to domain.InterviewRepository.
type InterviewRepoAdapter struct {
	repo *PostgresRepository
}

func NewInterviewRepoAdapter(repo *PostgresRepository) *InterviewRepoAdapter {
	return &InterviewRepoAdapter{repo: repo}
}

func (a *InterviewRepoAdapter) Create(ctx context.Context, interview *domain.Interview) error {
	return a.repo.CreateInterview(ctx, interview)
}

func (a *InterviewRepoAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Interview, error) {
	return a.repo.GetInterviewByID(ctx, id)
}

func (a *InterviewRepoAdapter) GetByInterviewerID(ctx context.Context, interviewerID uuid.UUID, limit, offset int) ([]*domain.Interview, error) {
	return a.repo.GetInterviewsByInterviewerID(ctx, interviewerID, limit, offset)
}

func (a *InterviewRepoAdapter) GetByCandidateID(ctx context.Context, candidateID uuid.UUID, limit, offset int) ([]*domain.Interview, error) {
	return a.repo.GetInterviewsByCandidateID(ctx, candidateID, limit, offset)
}

func (a *InterviewRepoAdapter) Update(ctx context.Context, interview *domain.Interview) error {
	return a.repo.UpdateInterview(ctx, interview)
}

func (a *InterviewRepoAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeleteInterview(ctx, id)
}

func (a *InterviewRepoAdapter) List(ctx context.Context, status *domain.InterviewStatus, limit, offset int) ([]*domain.Interview, error) {
	return a.repo.ListInterviews(ctx, status, limit, offset)
}

// QuestionRepoAdapter adapts PostgresRepository to domain.QuestionRepository.
type QuestionRepoAdapter struct {
	repo *PostgresRepository
}

func NewQuestionRepoAdapter(repo *PostgresRepository) *QuestionRepoAdapter {
	return &QuestionRepoAdapter{repo: repo}
}

func (a *QuestionRepoAdapter) Create(ctx context.Context, question *domain.Question) error {
	return a.repo.CreateQuestion(ctx, question)
}

func (a *QuestionRepoAdapter) CreateBatch(ctx context.Context, questions []*domain.Question) error {
	return a.repo.CreateQuestionsBatch(ctx, questions)
}

func (a *QuestionRepoAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Question, error) {
	return a.repo.GetQuestionByID(ctx, id)
}

func (a *QuestionRepoAdapter) GetByInterviewID(ctx context.Context, interviewID uuid.UUID) ([]*domain.Question, error) {
	return a.repo.GetQuestionsByInterviewID(ctx, interviewID)
}

func (a *QuestionRepoAdapter) Update(ctx context.Context, question *domain.Question) error {
	return a.repo.UpdateQuestion(ctx, question)
}

func (a *QuestionRepoAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeleteQuestion(ctx, id)
}

// SessionRepoAdapter adapts PostgresRepository to domain.SessionRepository.
type SessionRepoAdapter struct {
	repo *PostgresRepository
}

func NewSessionRepoAdapter(repo *PostgresRepository) *SessionRepoAdapter {
	return &SessionRepoAdapter{repo: repo}
}

func (a *SessionRepoAdapter) Create(ctx context.Context, session *domain.Session) error {
	return a.repo.CreateSession(ctx, session)
}

func (a *SessionRepoAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	return a.repo.GetSessionByID(ctx, id)
}

func (a *SessionRepoAdapter) GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*domain.Session, error) {
	return a.repo.GetSessionByInterviewID(ctx, interviewID)
}

func (a *SessionRepoAdapter) Update(ctx context.Context, session *domain.Session) error {
	return a.repo.UpdateSession(ctx, session)
}

func (a *SessionRepoAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeleteSession(ctx, id)
}

// AnswerRepoAdapter adapts PostgresRepository to domain.AnswerRepository.
type AnswerRepoAdapter struct {
	repo *PostgresRepository
}

func NewAnswerRepoAdapter(repo *PostgresRepository) *AnswerRepoAdapter {
	return &AnswerRepoAdapter{repo: repo}
}

func (a *AnswerRepoAdapter) Create(ctx context.Context, answer *domain.Answer) error {
	return a.repo.CreateAnswer(ctx, answer)
}

func (a *AnswerRepoAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Answer, error) {
	return a.repo.GetAnswerByID(ctx, id)
}

func (a *AnswerRepoAdapter) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.Answer, error) {
	return a.repo.GetAnswersBySessionID(ctx, sessionID)
}

func (a *AnswerRepoAdapter) Update(ctx context.Context, answer *domain.Answer) error {
	return a.repo.UpdateAnswer(ctx, answer)
}
