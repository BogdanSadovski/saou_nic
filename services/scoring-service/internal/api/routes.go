package api

import (
	"github.com/go-chi/chi/v5"
)

// registerRoutes configures all HTTP routes for the scoring service API.
func (h *Handler) registerRoutes() {
	// Health check
	h.router.Get("/health", h.healthCheck)

	// API v1 routes
	h.router.Route("/api/v1", func(r chi.Router) {
		// Interview module report endpoints
		r.Post("/scoring/generate", h.generateInterviewReport)
		r.Get("/scoring/reports/{session_id}", h.getInterviewReport)

		// Score endpoints
		r.Get("/scores", h.listScores)
		r.Post("/scores/evaluate", h.evaluateScore)
		r.Get("/scores/{id}", h.getScore)
		r.Delete("/scores/{id}", h.deleteScore)

		// Submission endpoints
		r.Get("/submissions/{submission_id}/scores", h.getScoresBySubmission)
		r.Get("/submissions/{submission_id}/result", h.getEvaluationResult)

		// Rubric endpoints
		r.Post("/rubrics", h.createRubric)
	})
}
