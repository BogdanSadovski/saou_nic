package api

import (
	"net/http"

	"github.com/interview-platform/interview-service/internal/websocket"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Router holds all routes for the API
type Router struct {
	*mux.Router
	handler        *Handler
	wsHandler      *websocket.Handler
	authMiddleware *AuthMiddleware
	logger         *logrus.Logger
}

// NewRouter creates and configures all routes
func NewRouter(
	handler *Handler,
	wsHandler *websocket.Handler,
	authMiddleware *AuthMiddleware,
	logger *logrus.Logger,
) *Router {
	r := &Router{
		Router:         mux.NewRouter(),
		handler:        handler,
		wsHandler:      wsHandler,
		authMiddleware: authMiddleware,
		logger:         logger,
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	// Health check (no auth required)
	r.HandleFunc("/health", r.handler.HealthCheck).Methods(http.MethodGet)

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()
	// Legacy API routes (kept for frontend compatibility)
	legacyAPI := r.PathPrefix("/api").Subrouter()

	// Public endpoints (no auth)
	api.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"message": "pong"})
	}).Methods(http.MethodGet)

	// Protected endpoints
	protected := api.PathPrefix("").Subrouter()
	protected.Use(r.authMiddleware.Authenticate)

	// AI Interview Module routes (JWT required)
	moduleSessions := protected.PathPrefix("/interviews/sessions").Subrouter()
	moduleSessions.HandleFunc("", r.handler.CreateInterviewModuleSession).Methods(http.MethodPost)
	moduleSessions.HandleFunc("/{session_id}", r.handler.GetInterviewModuleSession).Methods(http.MethodGet)
	moduleSessions.HandleFunc("/{session_id}/messages", r.handler.GetInterviewModuleMessages).Methods(http.MethodGet)
	moduleSessions.HandleFunc("/{session_id}/messages", r.handler.AddInterviewModuleMessage).Methods(http.MethodPost)
	moduleSessions.HandleFunc("/{session_id}/finish", r.handler.FinishInterviewModuleSession).Methods(http.MethodPost)
	moduleSessions.HandleFunc("/{session_id}/report", r.handler.GetInterviewModuleReport).Methods(http.MethodGet)
	moduleSessions.HandleFunc("/{session_id}/ws", r.handler.HandleInterviewModuleWS).Methods(http.MethodGet)

	// Interview routes
	interviews := protected.PathPrefix("/interviews").Subrouter()
	interviews.HandleFunc("/me/report", r.handler.GetMyInterviewAnalyticsReport).Methods(http.MethodGet)
	interviews.HandleFunc("", r.handler.CreateInterview).Methods(http.MethodPost)
	interviews.HandleFunc("", r.handler.ListInterviews).Methods(http.MethodGet)
	interviews.HandleFunc("/{id}", r.handler.GetInterview).Methods(http.MethodGet)
	interviews.HandleFunc("/{id}", r.handler.UpdateInterview).Methods(http.MethodPut)
	interviews.HandleFunc("/{id}", r.handler.DeleteInterview).Methods(http.MethodDelete)
	interviews.HandleFunc("/{id}/cancel", r.handler.CancelInterview).Methods(http.MethodPost)

	// Session routes
	sessions := protected.PathPrefix("/sessions").Subrouter()
	sessions.HandleFunc("/interview/{id}/start", r.handler.StartSession).Methods(http.MethodPost)
	sessions.HandleFunc("/{session_id}", r.handler.GetSession).Methods(http.MethodGet)
	sessions.HandleFunc("/{session_id}/end", r.handler.EndSession).Methods(http.MethodPost)
	sessions.HandleFunc("/{session_id}/results", r.handler.GetSessionResults).Methods(http.MethodGet)

	// Answer routes
	answers := protected.PathPrefix("/answers").Subrouter()
	answers.HandleFunc("/{session_id}/submit", r.handler.SubmitAnswer).Methods(http.MethodPost)

	// GitHub profile analytics routes
	github := protected.PathPrefix("/github").Subrouter()
	github.HandleFunc("/import", r.handler.ImportGitHubProfile).Methods(http.MethodPost)

	// Resume import analytics routes
	resume := protected.PathPrefix("/resume").Subrouter()
	resume.HandleFunc("/import", r.handler.ImportResumeProfile).Methods(http.MethodPost)
	resume.HandleFunc("/history", r.handler.GetResumeImportHistory).Methods(http.MethodGet)
	resume.HandleFunc("/history/{report_id}", r.handler.GetResumeImportReport).Methods(http.MethodGet)
	resume.HandleFunc("/vacancies/{report_id}", r.handler.GetMatchingVacancies).Methods(http.MethodGet)
	resume.HandleFunc("/devby/{report_id}", r.handler.GetMatchingDevByVacancies).Methods(http.MethodGet)

	// Legacy protected endpoints
	legacyProtected := legacyAPI.PathPrefix("").Subrouter()
	legacyProtected.Use(r.authMiddleware.Authenticate)
	legacyGithub := legacyProtected.PathPrefix("/github").Subrouter()
	legacyGithub.HandleFunc("/import", r.handler.ImportGitHubProfile).Methods(http.MethodPost)
	legacyResume := legacyProtected.PathPrefix("/resume").Subrouter()
	legacyResume.HandleFunc("/import", r.handler.ImportResumeProfile).Methods(http.MethodPost)
	legacyResume.HandleFunc("/history", r.handler.GetResumeImportHistory).Methods(http.MethodGet)
	legacyResume.HandleFunc("/history/{report_id}", r.handler.GetResumeImportReport).Methods(http.MethodGet)
	legacyResume.HandleFunc("/vacancies/{report_id}", r.handler.GetMatchingVacancies).Methods(http.MethodGet)
	legacyResume.HandleFunc("/devby/{report_id}", r.handler.GetMatchingDevByVacancies).Methods(http.MethodGet)

	// WebSocket endpoint
	r.HandleFunc("/ws", r.wsHandler.HandleWebSocket)
}

// ApplyMiddleware applies global middleware to the router
func (r *Router) ApplyMiddleware() {
	// Apply global middleware
	r.Use(mux.MiddlewareFunc(CORS()))
	r.Use(mux.MiddlewareFunc(Recovery(r.logger)))
	r.Use(mux.MiddlewareFunc(Logging(r.logger)))
}
