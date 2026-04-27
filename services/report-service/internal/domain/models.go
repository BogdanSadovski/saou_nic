package domain

import (
	"time"
)

// ReportStatus represents the lifecycle state of a report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusGenerating ReportStatus = "generating"
	ReportStatusCompleted ReportStatus = "completed"
	ReportStatusFailed    ReportStatus = "failed"
	ReportStatusExpired   ReportStatus = "expired"
)

// ReportFormat represents the output format of a report
type ReportFormat string

const (
	ReportFormatPDF  ReportFormat = "pdf"
	ReportFormatDOCX ReportFormat = "docx"
)

// ReportType represents the type of report to generate
type ReportType string

const (
	ReportTypeInterviewReport     ReportType = "interview_report"
	ReportTypeCandidateSummary    ReportType = "candidate_summary"
	ReportTypeAssessmentReport    ReportType = "assessment_report"
	ReportTypeComparativeAnalysis ReportType = "comparative_analysis"
)

// Report represents a generated report record
type Report struct {
	ID             string       `json:"id"`
	CandidateID    string       `json:"candidate_id"`
	InterviewID    string       `json:"interview_id,omitempty"`
	AssessmentID   string       `json:"assessment_id,omitempty"`
	Type           ReportType   `json:"type"`
	Format         ReportFormat `json:"format"`
	Status         ReportStatus `json:"status"`
	Title          string       `json:"title"`
	Description    string       `json:"description,omitempty"`
	FileURL        string       `json:"file_url,omitempty"`
	FileName       string       `json:"file_name,omitempty"`
	FileSize       int64        `json:"file_size,omitempty"`
	ErrorMessage   string       `json:"error_message,omitempty"`
	Metadata       ReportMetadata `json:"metadata,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	ExpiresAt      time.Time    `json:"expires_at,omitempty"`
	GeneratedBy    string       `json:"generated_by"`
}

// ReportMetadata holds additional data for report generation
type ReportMetadata struct {
	TemplateName    string            `json:"template_name,omitempty"`
	IncludeScores   bool              `json:"include_scores"`
	IncludeFeedback bool              `json:"include_feedback"`
	IncludeTimeline bool              `json:"include_timeline"`
	CustomFields    map[string]string `json:"custom_fields,omitempty"`
}

// CreateReportRequest represents the request to create a new report
type CreateReportRequest struct {
	CandidateID  string         `json:"candidate_id" binding:"required"`
	InterviewID  string         `json:"interview_id"`
	AssessmentID string         `json:"assessment_id"`
	Type         ReportType     `json:"type" binding:"required"`
	Format       ReportFormat   `json:"format"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Metadata     ReportMetadata `json:"metadata"`
}

// GenerateReportRequest represents the request to trigger report generation
type GenerateReportRequest struct {
	TemplateData map[string]interface{} `json:"template_data"`
}

// ReportListResponse represents a paginated list of reports
type ReportListResponse struct {
	Reports      []Report `json:"reports"`
	Total        int64    `json:"total"`
	Page         int      `json:"page"`
	PageSize     int      `json:"page_size"`
	HasMore      bool     `json:"has_more"`
}

// ReportStats holds statistics about report generation
type ReportStats struct {
	TotalReports      int64            `json:"total_reports"`
	ByStatus          map[string]int64 `json:"by_status"`
	ByFormat          map[string]int64 `json:"by_format"`
	ByType            map[string]int64 `json:"by_type"`
	AvgGenerationTime float64          `json:"avg_generation_time_ms"`
}
