package domain

import (
	"context"
)

// ReportRepository defines the interface for report data access
type ReportRepository interface {
	// Create inserts a new report record
	Create(ctx context.Context, report *Report) error

	// GetByID retrieves a report by its ID
	GetByID(ctx context.Context, id string) (*Report, error)

	// List retrieves reports with pagination and optional filters
	List(ctx context.Context, params ListReportsParams) (*ReportListResponse, error)

	// Update updates an existing report record
	Update(ctx context.Context, report *Report) error

	// UpdateStatus updates only the status and error message of a report
	UpdateStatus(ctx context.Context, id string, status ReportStatus, errorMsg string) error

	// Delete removes a report record
	Delete(ctx context.Context, id string) error

	// DeleteExpired removes all reports past their expiration date
	DeleteExpired(ctx context.Context) (int64, error)

	// GetStats returns aggregated report statistics
	GetStats(ctx context.Context) (*ReportStats, error)

	// GetByCandidateID retrieves all reports for a specific candidate
	GetByCandidateID(ctx context.Context, candidateID string) ([]Report, error)

	// GetPendingReports retrieves reports awaiting generation
	GetPendingReports(ctx context.Context) ([]Report, error)
}

// ListReportsParams holds parameters for listing reports
type ListReportsParams struct {
	Status      *ReportStatus
	Format      *ReportFormat
	Type        *ReportType
	CandidateID string
	Page        int
	PageSize    int
}

// FileStorage defines the interface for storing generated report files
type FileStorage interface {
	// Upload stores a file and returns its URL
	Upload(ctx context.Context, fileName string, contentType string, data []byte) (string, error)

	// Download retrieves a file by its URL
	Download(ctx context.Context, url string) ([]byte, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, url string) error

	// Exists checks if a file exists in storage
	Exists(ctx context.Context, url string) (bool, error)

	// GeneratePresignedURL creates a time-limited access URL
	GeneratePresignedURL(ctx context.Context, key string, expiresInSeconds int) (string, error)
}

// TemplateRepository defines the interface for accessing report templates
type TemplateRepository interface {
	// GetTemplate retrieves a template by name
	GetTemplate(name string) ([]byte, error)

	// ListTemplates returns all available template names
	ListTemplates() ([]string, error)
}
