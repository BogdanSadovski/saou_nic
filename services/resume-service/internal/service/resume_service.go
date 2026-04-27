package service

import (
	"context"
	"fmt"
	"time"

	"resume-service/internal/domain"
	"resume-service/internal/nlp"

	"github.com/google/uuid"
)

// ResumeService handles resume business logic
type ResumeService struct {
	repo       domain.ResumeRepository
	storage    domain.FileStorage
	nlpAnalyzer *NLPAnalyzer
}

// NewResumeService creates a new ResumeService
func NewResumeService(repo domain.ResumeRepository, storage domain.FileStorage, nlpAnalyzer *NLPAnalyzer) *ResumeService {
	return &ResumeService{
		repo:       repo,
		storage:    storage,
		nlpAnalyzer: nlpAnalyzer,
	}
}

// CreateResume creates a new resume from uploaded file data
func (s *ResumeService) CreateResume(ctx context.Context, input *domain.CreateResumeInput) (*domain.Resume, error) {
	// Generate unique ID
	id := uuid.New().String()

	// Generate storage key
	key := fmt.Sprintf("resumes/%s/%s", input.UserID, id)

	// Upload file to storage
	fileURL, err := s.storage.Upload(ctx, key, input.FileData, input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create resume record with pending status
	resume := &domain.Resume{
		ID:          id,
		UserID:      input.UserID,
		FileName:    input.FileName,
		FileURL:     fileURL,
		ContentType: input.ContentType,
		Status:      domain.StatusPending,
	}

	if err := s.repo.Create(ctx, resume); err != nil {
		return nil, fmt.Errorf("failed to create resume record: %w", err)
	}

	// Trigger async processing
	go s.processResume(ctx, resume, input.FileData)

	return resume, nil
}

// GetResume retrieves a resume by ID
func (s *ResumeService) GetResume(ctx context.Context, id string) (*domain.Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get resume: %w", err)
	}

	return resume, nil
}

// GetUserResumes retrieves all resumes for a user
func (s *ResumeService) GetUserResumes(ctx context.Context, userID string, limit, offset int) ([]*domain.Resume, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	resumes, err := s.repo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user resumes: %w", err)
	}

	return resumes, nil
}

// UpdateResume updates an existing resume
func (s *ResumeService) UpdateResume(ctx context.Context, input *domain.UpdateResumeInput) (*domain.Resume, error) {
	resume, err := s.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resume: %w", err)
	}

	if resume.UserID != input.UserID {
		return nil, fmt.Errorf("user does not have permission to update this resume")
	}

	// Apply updates
	if input.FirstName != nil {
		resume.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		resume.LastName = *input.LastName
	}
	if input.Email != nil {
		resume.Email = *input.Email
	}
	if input.Phone != nil {
		resume.Phone = *input.Phone
	}
	if input.Summary != nil {
		resume.Summary = *input.Summary
	}
	if input.Skills != nil {
		resume.Skills = *input.Skills
	}
	if input.Experience != nil {
		resume.Experience = *input.Experience
	}
	if input.Education != nil {
		resume.Education = *input.Education
	}
	if input.Languages != nil {
		resume.Languages = *input.Languages
	}
	if input.Certifications != nil {
		resume.Certifications = *input.Certifications
	}
	if input.Status != nil {
		resume.Status = *input.Status
	}

	if err := s.repo.Update(ctx, resume); err != nil {
		return nil, fmt.Errorf("failed to update resume: %w", err)
	}

	return resume, nil
}

// DeleteResume removes a resume and its associated file
func (s *ResumeService) DeleteResume(ctx context.Context, id, userID string) error {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get resume: %w", err)
	}

	if resume.UserID != userID {
		return fmt.Errorf("user does not have permission to delete this resume")
	}

	// Delete file from storage
	if err := s.storage.Delete(ctx, resume.FileURL); err != nil {
		// Log but continue - we still want to delete the DB record
		// In production, this should be handled with a saga or compensating transaction
	}

	// Delete DB record
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete resume: %w", err)
	}

	return nil
}

// ListResumes lists resumes with filtering
func (s *ResumeService) ListResumes(ctx context.Context, filter *domain.ResumeFilter) ([]*domain.Resume, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	resumes, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list resumes: %w", err)
	}

	return resumes, total, nil
}

// processResume handles the async parsing and analysis of a resume
func (s *ResumeService) processResume(ctx context.Context, resume *domain.Resume, fileData []byte) {
	// Update status to processing
	_ = s.repo.UpdateStatus(ctx, resume.ID, domain.StatusProcessing, "")

	// Parse the resume content
	parsedData, err := s.parseResume(fileData, resume.ContentType)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, resume.ID, domain.StatusFailed, fmt.Sprintf("parsing error: %v", err))
		return
	}

	// Extract entities using NLP
	entities, err := s.nlpAnalyzer.Analyze(ctx, parsedData.Text)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, resume.ID, domain.StatusFailed, fmt.Sprintf("NLP analysis error: %v", err))
		return
	}

	// Update resume with extracted data
	resume.FirstName = entities.FirstName
	resume.LastName = entities.LastName
	resume.Email = entities.Email
	resume.Phone = entities.Phone
	resume.Summary = entities.Summary
	resume.Skills = entities.Skills
	resume.Experience = entities.Experience
	resume.Education = entities.Education
	resume.Languages = entities.Languages
	resume.Certifications = entities.Certifications

	_ = s.repo.UpdateStatus(ctx, resume.ID, domain.StatusCompleted, "")
}

// parseResume extracts text content from different file formats
func (s *ResumeService) parseResume(fileData []byte, contentType string) (*nlp.ParsedResume, error) {
	switch contentType {
	case "application/pdf":
		return nlp.ParsePDF(fileData)
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return nlp.ParseDOCX(fileData)
	case "text/plain", "application/rtf":
		return nlp.ParsePlainText(fileData)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}

// GetResumeFileURL generates a presigned URL for downloading the resume file
func (s *ResumeService) GetResumeFileURL(ctx context.Context, id, userID string, expiresIn int) (string, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get resume: %w", err)
	}

	if resume.UserID != userID {
		return "", fmt.Errorf("user does not have permission to access this resume")
	}

	return s.storage.GetURL(ctx, resume.FileURL, expiresIn)
}

// ReparseResume triggers re-parsing of an existing resume
func (s *ResumeService) ReparseResume(ctx context.Context, id, userID string) error {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get resume: %w", err)
	}

	if resume.UserID != userID {
		return fmt.Errorf("user does not have permission to reparse this resume")
	}

	// Download file data
	fileData, err := s.storage.Download(ctx, resume.FileURL)
	if err != nil {
		return fmt.Errorf("failed to download resume file: %w", err)
	}

	// Reset status and trigger reprocessing
	_ = s.repo.UpdateStatus(ctx, id, domain.StatusPending, "")
	go s.processResume(ctx, resume, fileData)

	return nil
}

// GetResumeStats returns statistics about resume processing
func (s *ResumeService) GetResumeStats(ctx context.Context) (map[string]interface{}, error) {
	// Placeholder for statistics aggregation
	// In production, this would query the database for aggregated metrics
	return map[string]interface{}{
		"total_resumes":    0,
		"completed":        0,
		"failed":           0,
		"avg_processing_ms": 0,
		"last_updated":     time.Now().UTC(),
	}, nil
}
