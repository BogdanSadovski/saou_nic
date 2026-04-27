package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"analytics-service/internal/domain"
)

// ExportService handles exporting analytics data to various formats.
type ExportService struct {
	exportRepo  domain.ExportRepository
	eventRepo   domain.EventRepository
	maxPageSize int
}

// NewExportService creates a new ExportService.
func NewExportService(
	exportRepo domain.ExportRepository,
	eventRepo domain.EventRepository,
	maxPageSize int,
) *ExportService {
	return &ExportService{
		exportRepo:  exportRepo,
		eventRepo:   eventRepo,
		maxPageSize: maxPageSize,
	}
}

// CreateExportRequest initializes a new export request.
func (s *ExportService) CreateExportRequest(
	ctx context.Context,
	tenantID, format string,
	filter domain.QueryFilter,
) (*domain.ExportRequest, error) {
	req := &domain.ExportRequest{
		TenantID:  tenantID,
		Format:    format,
		Filter:    filter,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.exportRepo.CreateExport(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create export request: %w", err)
	}

	return req, nil
}

// ProcessExport generates the export file and updates the request status.
func (s *ExportService) ProcessExport(ctx context.Context, exportID string) error {
	req, err := s.exportRepo.GetExport(ctx, exportID)
	if err != nil {
		return fmt.Errorf("failed to get export request: %w", err)
	}

	if req.Status != "pending" {
		return fmt.Errorf("export request %s is not pending (status: %s)", exportID, req.Status)
	}

	// Update status to processing.
	req.Status = "processing"
	req.UpdatedAt = time.Now().UTC()
	if err := s.exportRepo.UpdateExport(ctx, req); err != nil {
		return fmt.Errorf("failed to update export status: %w", err)
	}

	// Fetch data.
	limit := req.Filter.Limit
	if limit <= 0 || limit > s.maxPageSize {
		limit = s.maxPageSize
	}
	req.Filter.Limit = limit

	events, err := s.eventRepo.Query(ctx, req.Filter)
	if err != nil {
		s.failExport(ctx, req, fmt.Sprintf("query failed: %v", err))
		return err
	}

	// Generate file based on format.
	var fileURL string
	switch strings.ToLower(req.Format) {
	case "csv":
		fileURL, err = s.exportToCSV(ctx, events, req.TenantID)
	case "json":
		fileURL, err = s.exportToJSON(ctx, events, req.TenantID)
	default:
		s.failExport(ctx, req, fmt.Sprintf("unsupported format: %s", req.Format))
		return fmt.Errorf("unsupported export format: %s", req.Format)
	}

	if err != nil {
		s.failExport(ctx, req, fmt.Sprintf("export failed: %v", err))
		return err
	}

	// Mark as completed.
	req.Status = "completed"
	req.FileURL = fileURL
	req.UpdatedAt = time.Now().UTC()
	if err := s.exportRepo.UpdateExport(ctx, req); err != nil {
		return fmt.Errorf("failed to finalize export: %w", err)
	}

	return nil
}

// GetExportStatus returns the current status of an export request.
func (s *ExportService) GetExportStatus(ctx context.Context, exportID string) (*domain.ExportRequest, error) {
	req, err := s.exportRepo.GetExport(ctx, exportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get export request: %w", err)
	}

	return req, nil
}

// ListExports returns export requests for a tenant.
func (s *ExportService) ListExports(
	ctx context.Context,
	tenantID string,
	limit, offset int,
) ([]*domain.ExportRequest, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > s.maxPageSize {
		limit = s.maxPageSize
	}

	exports, err := s.exportRepo.ListExports(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list exports: %w", err)
	}

	return exports, nil
}

// GetSupportedFormats returns the list of available export formats.
func (s *ExportService) GetSupportedFormats() []string {
	return []string{"csv", "json"}
}

func (s *ExportService) exportToCSV(ctx context.Context, events []*domain.Event, tenantID string) (string, error) {
	_ = ctx // used for storage integration

	if len(events) == 0 {
		return "", fmt.Errorf("no events to export")
	}

	// Build CSV in memory.
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header.
	header := []string{"id", "type", "user_id", "session_id", "url", "timestamp", "country", "device", "os", "browser"}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows.
	for _, e := range events {
		row := []string{
			e.ID,
			string(e.Type),
			e.UserID,
			e.SessionID,
			e.URL,
			e.Timestamp.Format(time.RFC3339),
			e.Country,
			e.Device,
			e.OS,
			e.Browser,
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	// In production, upload to S3/object storage and return URL.
	// For now, return an in-memory placeholder path.
	filePath := fmt.Sprintf("/exports/%s/export_%s.csv", tenantID, time.Now().Format("20060102_150405"))
	_ = buf.String() // CSV content would be uploaded

	return filePath, nil
}

func (s *ExportService) exportToJSON(ctx context.Context, events []*domain.Event, tenantID string) (string, error) {
	_ = ctx

	if len(events) == 0 {
		return "", fmt.Errorf("no events to export")
	}

	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(events); err != nil {
		return "", fmt.Errorf("failed to encode events to JSON: %w", err)
	}

	filePath := fmt.Sprintf("/exports/%s/export_%s.json", tenantID, time.Now().Format("20060102_150405"))
	_ = buf.String()

	return filePath, nil
}

func (s *ExportService) failExport(ctx context.Context, req *domain.ExportRequest, errMsg string) {
	req.Status = "failed"
	req.Error = errMsg
	req.UpdatedAt = time.Now().UTC()
	_ = s.exportRepo.UpdateExport(ctx, req)
}

// ExportWriter wraps io.Writer for streaming exports.
type ExportWriter struct {
	writer io.Writer
	format string
}

// NewExportWriter creates a streaming export writer.
func NewExportWriter(w io.Writer, format string) *ExportWriter {
	return &ExportWriter{writer: w, format: format}
}

// WriteHeader writes the appropriate header for the format.
func (ew *ExportWriter) WriteHeader() error {
	if ew.format == "json" {
		_, err := ew.writer.Write([]byte("[\n"))
		return err
	}
	// CSV header is written by the export function.
	return nil
}

// Close finalizes the export output.
func (ew *ExportWriter) Close() error {
	if ew.format == "json" {
		_, err := ew.writer.Write([]byte("\n]\n"))
		return err
	}
	return nil
}
