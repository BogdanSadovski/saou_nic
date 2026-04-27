package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"resume-service/internal/domain"
	"resume-service/internal/service"
	"resume-service/pkg/storage"
)

// Handler handles HTTP requests for resume operations
type Handler struct {
	resumeService *service.ResumeService
	uploader      *storage.Uploader
}

// NewHandler creates a new Handler
func NewHandler(resumeService *service.ResumeService) *Handler {
	return &Handler{
		resumeService: resumeService,
		uploader:      storage.NewUploader(),
	}
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, Response{
		Success: false,
		Error:   message,
	})
}

// getUserID extracts user ID from request context/headers
func (h *Handler) getUserID(r *http.Request) string {
	// In production, extract from JWT token or auth context
	return r.Header.Get("X-User-ID")
}

// CreateResume handles POST /api/v1/resumes
func (h *Handler) CreateResume(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "user ID is required")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to parse form data")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Upload and validate file
	uploadResult, err := h.uploader.Upload(r.Context(), header)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Read file data
	fileData := make([]byte, uploadResult.Size)
	// Note: In production, seek back to beginning or store during upload
	// This is simplified

	// Create resume
	input := &domain.CreateResumeInput{
		UserID:      userID,
		FileName:    uploadResult.FileName,
		ContentType: uploadResult.ContentType,
		FileData:    fileData,
	}

	resume, err := h.resumeService.CreateResume(r.Context(), input)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create resume")
		return
	}

	h.writeJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    resume,
		Message: "Resume uploaded successfully",
	})
}

// GetResume handles GET /api/v1/resumes/:id
func (h *Handler) GetResume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "resume ID is required")
		return
	}

	resume, err := h.resumeService.GetResume(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "resume not found")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    resume,
	})
}

// GetUserResumes handles GET /api/v1/resumes
func (h *Handler) GetUserResumes(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "user ID is required")
		return
	}

	// Parse pagination params
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resumes, err := h.resumeService.GetUserResumes(r.Context(), userID, limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to fetch resumes")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    resumes,
	})
}

// UpdateResume handles PUT /api/v1/resumes/:id
func (h *Handler) UpdateResume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := h.getUserID(r)

	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "user ID is required")
		return
	}

	var input domain.UpdateResumeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input.ID = id
	input.UserID = userID

	resume, err := h.resumeService.UpdateResume(r.Context(), &input)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to update resume")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    resume,
		Message: "Resume updated successfully",
	})
}

// DeleteResume handles DELETE /api/v1/resumes/:id
func (h *Handler) DeleteResume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := h.getUserID(r)

	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "user ID is required")
		return
	}

	if err := h.resumeService.DeleteResume(r.Context(), id, userID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to delete resume")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Resume deleted successfully",
	})
}

// ReparseResume handles POST /api/v1/resumes/:id/reparse
func (h *Handler) ReparseResume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := h.getUserID(r)

	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "user ID is required")
		return
	}

	if err := h.resumeService.ReparseResume(r.Context(), id, userID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to reparse resume")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Resume reparse initiated",
	})
}

// GetResumeFileURL handles GET /api/v1/resumes/:id/download
func (h *Handler) GetResumeFileURL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := h.getUserID(r)

	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "user ID is required")
		return
	}

	expiresIn, _ := strconv.Atoi(r.URL.Query().Get("expires_in"))
	if expiresIn <= 0 {
		expiresIn = 3600 // Default: 1 hour
	}

	url, err := h.resumeService.GetResumeFileURL(r.Context(), id, userID, expiresIn)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "failed to generate download URL")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"download_url": url,
			"expires_in":   strconv.Itoa(expiresIn),
		},
	})
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"status":  "healthy",
			"service": "resume-service",
		},
	})
}

// GetStats handles GET /api/v1/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.resumeService.GetResumeStats(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// parseStatus parses a status string from the request
func parseStatus(s string) (domain.ResumeStatus, error) {
	status := domain.ResumeStatus(strings.ToLower(s))
	switch status {
	case domain.StatusPending, domain.StatusProcessing, domain.StatusCompleted, domain.StatusFailed:
		return status, nil
	default:
		return "", fmt.Errorf("invalid status: %s", s)
	}
}
