package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bogdan/real_ass/report-service/internal/domain"
	"github.com/bogdan/real_ass/report-service/internal/service"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	reportService *service.ReportService
	pdfService    *service.PDFGeneratorService
	docxService   *service.DOCXGeneratorService
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	reportService *service.ReportService,
	pdfService *service.PDFGeneratorService,
	docxService *service.DOCXGeneratorService,
) *Handlers {
	return &Handlers{
		reportService: reportService,
		pdfService:    pdfService,
		docxService:   docxService,
	}
}

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"-"`
}

// SuccessResponse represents a JSON success response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

// HandleCreateReport handles POST /api/v1/reports
func (h *Handlers) HandleCreateReport(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := h.reportService.ValidateReportRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	report, err := h.reportService.CreateReport(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create report: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, SuccessResponse{Data: report, Message: "Report created successfully"})
}

// HandleGetReport handles GET /api/v1/reports/{id}
func (h *Handlers) HandleGetReport(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "report ID is required")
		return
	}

	report, err := h.reportService.GetReport(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "report not found")
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch report: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Data: report})
}

// HandleListReports handles GET /api/v1/reports
func (h *Handlers) HandleListReports(w http.ResponseWriter, r *http.Request) {
	params := domain.ListReportsParams{
		Page:     1,
		PageSize: 20,
	}

	// Parse query parameters
	query := r.URL.Query()
	if status := query.Get("status"); status != "" {
		s := domain.ReportStatus(status)
		params.Status = &s
	}
	if format := query.Get("format"); format != "" {
		f := domain.ReportFormat(format)
		params.Format = &f
	}
	if reportType := query.Get("type"); reportType != "" {
		t := domain.ReportType(reportType)
		params.Type = &t
	}
	if candidateID := query.Get("candidate_id"); candidateID != "" {
		params.CandidateID = candidateID
	}
	if page := query.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
			params.PageSize = ps
		}
	}

	response, err := h.reportService.ListReports(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list reports: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Data: response})
}

// HandleDeleteReport handles DELETE /api/v1/reports/{id}
func (h *Handlers) HandleDeleteReport(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "report ID is required")
		return
	}

	if err := h.reportService.DeleteReport(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "report not found")
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete report: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGeneratePDF handles POST /api/v1/reports/{id}/generate/pdf
func (h *Handlers) HandleGeneratePDF(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "report ID is required")
		return
	}

	var req service.GenerateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.pdfService.GenerateReport(r.Context(), id, req.TemplateData); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate PDF: %v", err))
		return
	}

	report, err := h.reportService.GetReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch updated report: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Data: report, Message: "PDF generated successfully"})
}

// HandleGenerateDOCX handles POST /api/v1/reports/{id}/generate/docx
func (h *Handlers) HandleGenerateDOCX(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "report ID is required")
		return
	}

	var req service.GenerateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.docxService.GenerateReport(r.Context(), id, req.TemplateData); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate DOCX: %v", err))
		return
	}

	report, err := h.reportService.GetReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch updated report: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Data: report, Message: "DOCX generated successfully"})
}

// HandleDownloadReport handles GET /api/v1/reports/{id}/download
func (h *Handlers) HandleDownloadReport(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "report ID is required")
		return
	}

	report, err := h.reportService.GetReport(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "report not found")
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch report: %v", err))
		return
	}

	if report.Status != domain.ReportStatusCompleted {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("report is not ready for download (status: %s)", report.Status))
		return
	}

	if report.FileURL == "" {
		writeError(w, http.StatusNotFound, "report file not found")
		return
	}

	// Generate a presigned URL for download (valid for 1 hour)
	// Note: In a real implementation, you'd use the storage interface directly
	// For now, we redirect to the stored URL
	http.Redirect(w, r, report.FileURL, http.StatusSeeOther)
}

// HandleGetStats handles GET /api/v1/reports/stats
func (h *Handlers) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.reportService.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stats: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Data: stats})
}

// HandleHealthCheck handles GET /health
func (h *Handlers) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "report-service",
	})
}

// HandleGetReportsByCandidate handles GET /api/v1/candidates/{id}/reports
func (h *Handlers) HandleGetReportsByCandidate(w http.ResponseWriter, r *http.Request) {
	candidateID := extractCandidateIDFromPath(r.URL.Path)
	if candidateID == "" {
		writeError(w, http.StatusBadRequest, "candidate ID is required")
		return
	}

	reports, err := h.reportService.GetReportsByCandidate(r.Context(), candidateID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch reports: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Data: reports})
}

// extractIDFromPath extracts the report ID from the URL path
func extractIDFromPath(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	// Expected path: api/v1/reports/{id} or api/v1/reports/{id}/...
	for i, seg := range segments {
		if seg == "reports" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}

// extractCandidateIDFromPath extracts the candidate ID from the URL path
func extractCandidateIDFromPath(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	// Expected path: api/v1/candidates/{id}/reports
	for i, seg := range segments {
		if seg == "candidates" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}
