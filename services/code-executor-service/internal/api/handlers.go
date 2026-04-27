package api

import (
	"encoding/json"
	"log"
	"net/http"

	"code-executor-service/internal/config"
	"code-executor-service/internal/domain"
	"code-executor-service/internal/executor"
)

type Handler struct {
	executor *executor.Executor
	cfg      *config.Config
	logger   *log.Logger
}

func NewHandler(executor *executor.Executor, cfg *config.Config) *Handler {
	return &Handler{
		executor: executor,
		cfg:      cfg,
		logger:   log.Default(),
	}
}

func (h *Handler) ExecuteCode(w http.ResponseWriter, r *http.Request) {
	var req domain.CodeExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result := h.executor.Execute(r.Context(), &req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "code-executor-service",
	})
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/execute", h.ExecuteCode)
	mux.HandleFunc("/healthz", h.Healthz)
}
