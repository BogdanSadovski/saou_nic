package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/interview-platform/interview-service/internal/repository"
	"github.com/interview-platform/interview-service/pkg/codeexecutor"
)

// SubmitCodeRequest - API request for code submission
type SubmitCodeRequest struct {
	Language  string                  `json:"language"`
	Code      string                  `json:"code"`
	Input     string                  `json:"input,omitempty"`
	TestCases []codeexecutor.TestCase `json:"test_cases,omitempty"`
}

// SubmitCodeResponse - API response for code submission
type SubmitCodeResponse struct {
	Status       string               `json:"status"`
	SubmissionID string               `json:"submission_id"`
	Output       string               `json:"output"`
	Error        string               `json:"error,omitempty"`
	Runtime      time.Duration        `json:"runtime"`
	TestResults  []testResultResponse `json:"test_results,omitempty"`
	ExitCode     int                  `json:"exit_code"`
}

type testResultResponse struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

// SubmitCode handles code submission during interview
func (h *Handler) SubmitCode(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		h.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req SubmitCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Create code submission record
	submission := &repository.CodeSubmission{
		SessionID: sessionID,
		UserID:    userID,
		Language:  req.Language,
		Code:      req.Code,
		InputData: &req.Input,
	}

	if submission, err = h.repo.CreateCodeSubmission(r.Context(), submission); err != nil {
		h.logger.WithError(err).Error("failed to create code submission")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save submission"})
		return
	}

	// Execute code via code executor service
	execReq := &codeexecutor.CodeExecutionRequest{
		Language:  req.Language,
		Code:      req.Code,
		Input:     req.Input,
		TestCases: req.TestCases,
	}

	result, err := h.codeExecutor.Execute(r.Context(), execReq)
	if err != nil {
		h.logger.WithError(err).Error("code execution failed")
		h.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "code execution service unavailable"})
		return
	}

	// Store execution result
	testResultsJSON, _ := json.Marshal(result.TestResults)
	execResult := &repository.CodeExecutionResult{
		SubmissionID:    submission.ID,
		Status:          result.Status,
		Output:          &result.Output,
		ExecutionTimeMs: int64Ptr(result.Runtime.Milliseconds()),
		MemoryUsedBytes: int64Ptr(result.Memory),
		ExitCode:        intPtr(result.ExitCode),
		TestResults:     testResultsJSON,
	}

	if result.Error != "" {
		execResult.ErrorMessage = &result.Error
	}

	if _, err := h.repo.CreateCodeExecutionResult(r.Context(), execResult); err != nil {
		h.logger.WithError(err).Error("failed to store execution result")
		// Don't fail, just log
	}

	// Format response
	response := SubmitCodeResponse{
		Status:       result.Status,
		SubmissionID: submission.ID.String(),
		Output:       result.Output,
		Error:        result.Error,
		Runtime:      result.Runtime,
		ExitCode:     result.ExitCode,
	}

	// Convert test results to response format
	for _, tr := range result.TestResults {
		response.TestResults = append(response.TestResults, testResultResponse{
			Name:     tr.Name,
			Passed:   tr.Passed,
			Expected: tr.Expected,
			Actual:   tr.Actual,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetCodeSubmissions retrieves all code submissions for a session
func (h *Handler) GetCodeSubmissions(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	submissions, err := h.repo.ListCodeSubmissionsBySession(r.Context(), sessionID)
	if err != nil {
		h.logger.WithError(err).Error("failed to get code submissions")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch submissions"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": submissions,
		"count":       len(submissions),
	})
}

// Helper functions
func int64Ptr(v int64) *int64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func getUserIDFromContext(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
