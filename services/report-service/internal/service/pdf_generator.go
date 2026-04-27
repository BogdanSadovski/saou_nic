package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bogdan/real_ass/report-service/internal/domain"
	"github.com/bogdan/real_ass/report-service/pkg/generator"
)

// PDFGeneratorService handles PDF report generation workflow
type PDFGeneratorService struct {
	generator *generator.PDFGenerator
	repo      domain.ReportRepository
	storage   domain.FileStorage
}

// NewPDFGeneratorService creates a new PDF generator service
func NewPDFGeneratorService(gen *generator.PDFGenerator, repo domain.ReportRepository, storage domain.FileStorage) *PDFGeneratorService {
	return &PDFGeneratorService{
		generator: gen,
		repo:      repo,
		storage:   storage,
	}
}

// GenerateReport generates a PDF report for the given report ID
func (s *PDFGeneratorService) GenerateReport(ctx context.Context, reportID string, templateData map[string]interface{}) error {
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

	// Generate PDF
	startTime := time.Now()
	pdfData, err := s.generator.Generate(templateName, templateData)
	if err != nil {
		s.repo.UpdateStatus(ctx, reportID, domain.ReportStatusFailed, fmt.Sprintf("PDF generation failed: %v", err))
		return fmt.Errorf("generating PDF: %w", err)
	}

	generationTime := time.Since(startTime)
	_ = generationTime // Could be stored in metadata for analytics

	// Determine file name
	fileName := fmt.Sprintf("%s_%s.pdf", report.Type, report.ID[:8])

	// Upload to storage
	fileURL, err := s.storage.Upload(ctx, fileName, "application/pdf", pdfData)
	if err != nil {
		s.repo.UpdateStatus(ctx, reportID, domain.ReportStatusFailed, fmt.Sprintf("upload failed: %v", err))
		return fmt.Errorf("uploading PDF: %w", err)
	}

	// Update report with file info
	report.Status = domain.ReportStatusCompleted
	report.FileURL = fileURL
	report.FileName = fileName
	report.FileSize = int64(len(pdfData))
	report.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, report); err != nil {
		return fmt.Errorf("updating report: %w", err)
	}

	return nil
}

// GenerateReportFromData creates a new report and generates PDF in one operation
func (s *PDFGeneratorService) GenerateReportFromData(ctx context.Context, req domain.CreateReportRequest, templateData map[string]interface{}) (*domain.Report, error) {
	// Create report record
	reportService := &ReportService{repo: s.repo, storage: s.storage}
	report, err := reportService.CreateReport(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating report: %w", err)
	}

	// Generate PDF
	if err := s.GenerateReport(ctx, report.ID, templateData); err != nil {
		return report, fmt.Errorf("generating PDF: %w", err)
	}

	// Fetch updated report
	updatedReport, err := s.repo.GetByID(ctx, report.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching updated report: %w", err)
	}

	return updatedReport, nil
}

// GetDefaultTemplate returns the default template name for a report type
func (s *PDFGeneratorService) getDefaultTemplate(reportType domain.ReportType) string {
	switch reportType {
	case domain.ReportTypeInterviewReport:
		return "interview_report.pdf.tmpl"
	case domain.ReportTypeCandidateSummary:
		return "candidate_summary.pdf.tmpl"
	case domain.ReportTypeAssessmentReport:
		return "assessment_report.pdf.tmpl"
	case domain.ReportTypeComparativeAnalysis:
		return "comparative_analysis.pdf.tmpl"
	default:
		return "default.pdf.tmpl"
	}
}
