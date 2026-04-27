package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bogdan/real_ass/report-service/internal/domain"
	"github.com/bogdan/real_ass/report-service/pkg/generator"
)

// DOCXGeneratorService handles DOCX report generation workflow
type DOCXGeneratorService struct {
	generator *generator.DOCXGenerator
	repo      domain.ReportRepository
	storage   domain.FileStorage
}

// NewDOCXGeneratorService creates a new DOCX generator service
func NewDOCXGeneratorService(gen *generator.DOCXGenerator, repo domain.ReportRepository, storage domain.FileStorage) *DOCXGeneratorService {
	return &DOCXGeneratorService{
		generator: gen,
		repo:      repo,
		storage:   storage,
	}
}

// GenerateReport generates a DOCX report for the given report ID
func (s *DOCXGeneratorService) GenerateReport(ctx context.Context, reportID string, templateData map[string]interface{}) error {
	// Update status to generating
	if err := s.repo.UpdateStatus(ctx, reportID, domain.ReportStatusGenerating, ""); err != nil {
		return fmt.Errorf("updating report status: %w", err)
	}

	// Fetch the report
	report, err := s.repo.GetByID(ctx, reportID)
	if err != nil {
		s.repo.UpdateStatus(ctx, reportID, domain.ReportStatusFailed, fmt.Sprintf("failed to fetch report: %v", err))
		return fmt.Errorf("fetching report: %w", err)
	}

	// Determine template based on report type
	templateName := report.Metadata.TemplateName
	if templateName == "" {
		templateName = s.getDefaultTemplate(report.Type)
	}

	// Generate DOCX
	startTime := time.Now()
	docxData, err := s.generator.Generate(templateName, templateData)
	if err != nil {
		s.repo.UpdateStatus(ctx, reportID, domain.ReportStatusFailed, fmt.Sprintf("DOCX generation failed: %v", err))
		return fmt.Errorf("generating DOCX: %w", err)
	}

	generationTime := time.Since(startTime)
	_ = generationTime // Could be stored in metadata for analytics

	// Determine file name
	fileName := fmt.Sprintf("%s_%s.docx", report.Type, report.ID[:8])

	// Upload to storage
	fileURL, err := s.storage.Upload(ctx, fileName, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", docxData)
	if err != nil {
		s.repo.UpdateStatus(ctx, reportID, domain.ReportStatusFailed, fmt.Sprintf("upload failed: %v", err))
		return fmt.Errorf("uploading DOCX: %w", err)
	}

	// Update report with file info
	report.Status = domain.ReportStatusCompleted
	report.FileURL = fileURL
	report.FileName = fileName
	report.FileSize = int64(len(docxData))
	report.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, report); err != nil {
		return fmt.Errorf("updating report: %w", err)
	}

	return nil
}

// GenerateReportFromData creates a new report and generates DOCX in one operation
func (s *DOCXGeneratorService) GenerateReportFromData(ctx context.Context, req domain.CreateReportRequest, templateData map[string]interface{}) (*domain.Report, error) {
	// Create report record
	reportService := &ReportService{repo: s.repo, storage: s.storage}
	report, err := reportService.CreateReport(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating report: %w", err)
	}

	// Generate DOCX
	if err := s.GenerateReport(ctx, report.ID, templateData); err != nil {
		return report, fmt.Errorf("generating DOCX: %w", err)
	}

	// Fetch updated report
	updatedReport, err := s.repo.GetByID(ctx, report.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching updated report: %w", err)
	}

	return updatedReport, nil
}

// GetDefaultTemplate returns the default template name for a report type
func (s *DOCXGeneratorService) getDefaultTemplate(reportType domain.ReportType) string {
	switch reportType {
	case domain.ReportTypeInterviewReport:
		return "interview_report.docx.tmpl"
	case domain.ReportTypeCandidateSummary:
		return "candidate_summary.docx.tmpl"
	case domain.ReportTypeAssessmentReport:
		return "assessment_report.docx.tmpl"
	case domain.ReportTypeComparativeAnalysis:
		return "comparative_analysis.docx.tmpl"
	default:
		return "default.docx.tmpl"
	}
}
