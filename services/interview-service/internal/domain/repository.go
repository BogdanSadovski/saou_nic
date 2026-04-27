package domain

import (
	"context"

	"github.com/google/uuid"
)

type InterviewRepository interface {
	Create(ctx context.Context, interview *Interview) error
	GetByID(ctx context.Context, id uuid.UUID) (*Interview, error)
	GetByInterviewerID(ctx context.Context, interviewerID uuid.UUID, limit, offset int) ([]*Interview, error)
	GetByCandidateID(ctx context.Context, candidateID uuid.UUID, limit, offset int) ([]*Interview, error)
	Update(ctx context.Context, interview *Interview) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, status *InterviewStatus, limit, offset int) ([]*Interview, error)
}

type QuestionRepository interface {
	Create(ctx context.Context, question *Question) error
	CreateBatch(ctx context.Context, questions []*Question) error
	GetByID(ctx context.Context, id uuid.UUID) (*Question, error)
	GetByInterviewID(ctx context.Context, interviewID uuid.UUID) ([]*Question, error)
	Update(ctx context.Context, question *Question) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*Session, error)
	GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*Session, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AnswerRepository interface {
	Create(ctx context.Context, answer *Answer) error
	GetByID(ctx context.Context, id uuid.UUID) (*Answer, error)
	GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*Answer, error)
	Update(ctx context.Context, answer *Answer) error
}

type Repository struct {
	Interview InterviewRepository
	Question  QuestionRepository
	Session   SessionRepository
	Answer    AnswerRepository
}
