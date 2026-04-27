package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/bogdan/real_ass/report-service/internal/config"
	"github.com/bogdan/real_ass/report-service/internal/domain"
)

// ReportService handles report CRUD operations and business logic
type ReportService struct {
	repo   domain.ReportRepository
	storage domain.FileStorage
	cfg    config.ServiceConfig
}

// NewReportService creates a new report service
func NewReportService(repo domain.ReportRepository, storage domain.FileStorage, cfg config.ServiceConfig) *ReportService {
	return &ReportService{
		repo:    repo,
		storage: storage,
		cfg:     cfg,
	}
}

// CreateReport creates a new report record
func (s *ReportService) CreateReport(ctx context.Context, req domain.CreateReportRequest) (*domain.Report, error) {
	now := time.Now()

	report := &domain.Report{
		ID:          uuid.New().String(),
		CandidateID: req.CandidateID,
		InterviewID: req.InterviewID,
		AssessmentID: req.AssessmentID,
		Type:        req.Type,
		Format:      req.Format,
		Status:      domain.ReportStatusPending,
		Title:       req.Title,
		Description: req.Description,
		Metadata:    req.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.AddDate(0, 0, s.cfg.ReportRetentionDays),
		GeneratedBy: "system",
	}

	// Apply default format if not specified
	if report.Format == "" {
		report.Format = domain.ReportFormatPDF
	}

	if err := s.repo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("creating report record: %w", err)
	}

	return report, nil
}

// GetReport retrieves a report by ID
func (s *ReportService) GetReport(ctx context.Context, id string) (*domain.Report, error) {
	report, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching report: %w", err)
	}
	return report, nil
}

// ListReports retrieves reports with pagination and filters
func (s *ReportService) ListReports(ctx context.Context, params domain.ListReportsParams) (*domain.ReportListResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	response, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("listing reports: %w", err)
	}

	return response, nil
}

// DeleteReport deletes a report and its associated file
func (s *ReportService) DeleteReport(ctx context.Context, id string) error {
	report, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching report: %w", err)
	}

	// Delete the file from storage if it exists
	if report.FileURL != "" {
		if err := s.storage.Delete(ctx, report.FileURL); err != nil {
			// Log the error but continue with report deletion
			// In production, you might want to handle this differently
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting report: %w", err)
	}

	return nil
}

// GetReportsByCandidate retrieves all reports for a candidate
func (s *ReportService) GetReportsByCandidate(ctx context.Context, candidateID string) ([]domain.Report, error) {
	reports, err := s.repo.GetByCandidateID(ctx, candidateID)
	if err != nil {
		return nil, fmt.Errorf("fetching reports by candidate: %w", err)
	}
	return reports, nil
}

// GetStats returns report generation statistics
func (s *ReportService) GetStats(ctx context.Context) (*domain.ReportStats, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching report stats: %w", err)
	}
	return stats, nil
}

// CleanupExpiredReports removes all expired reports
func (s *ReportService) CleanupExpiredReports(ctx context.Context) (int64, error) {
	deleted, err := s.repo.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("deleting expired reports: %w", err)
	}
	return deleted, nil
}

// ValidateReportRequest validates a create report request
func (s *ReportService) ValidateReportRequest(req *domain.CreateReportRequest) error {
	if req.CandidateID == "" {
		return fmt.Errorf("candidate_id is required")
	}
	if req.Type == "" {
		return fmt.Errorf("report type is required")
	}

	// Validate format
	if req.Format != "" && req.Format != domain.ReportFormatPDF && req.Format != domain.ReportFormatDOCX {
		return fmt.Errorf("invalid format: %s, must be 'pdf' or 'docx'", req.Format)
	}

	// Validate type
	validTypes := map[domain.ReportType]bool{
		domain.ReportTypeInterviewReport:     true,
		domain.ReportTypeCandidateSummary:    true,
		domain.ReportTypeAssessmentReport:    true,
		domain.ReportTypeComparativeAnalysis: true,
	}
	if !validTypes[req.Type] {
		return fmt.Errorf("invalid report type: %s", req.Type)
	}

	return nil
}
