package api

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"scoring-service/internal/domain"
	"scoring-service/internal/service"

	"github.com/go-chi/chi/v5"
)

// Handler holds the dependencies for HTTP request handling.
type Handler struct {
	scoringService   *service.ScoringService
	router           *chi.Mux
	reportsMu        sync.RWMutex
	interviewReports map[string]interviewReport
}

// NewHandler creates a new HTTP handler with the scoring service.
func NewHandler(scoringService *service.ScoringService) *Handler {
	h := &Handler{
		scoringService:   scoringService,
		router:           chi.NewRouter(),
		interviewReports: make(map[string]interviewReport),
	}
	h.registerRoutes()
	return h
}

type interviewMessage struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
	// AI verdict on this user message — drives correctness/clarity
	// math. Empty / "none" means the LLM didn't grade this turn.
	Verdict string `json:"verdict,omitempty"`
}

type generateInterviewReportRequest struct {
	SessionID string             `json:"session_id"`
	Role      string             `json:"role"`
	Level     string             `json:"level"`
	Messages  []interviewMessage `json:"messages"`
	Feedback  string             `json:"feedback"`
}

type interviewReport struct {
	SessionID       string    `json:"session_id"`
	Correctness     float64   `json:"correctness"`
	Clarity         float64   `json:"clarity"`
	Completeness    float64   `json:"completeness"`
	Relevance       float64   `json:"relevance"`
	OverallScore    float64   `json:"overall_score"`
	Strengths       []string  `json:"strengths"`
	Weaknesses      []string  `json:"weaknesses"`
	Recommendations []string  `json:"recommendations"`
	GeneratedAt     time.Time `json:"generated_at"`
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

// evaluateScore handles POST /api/v1/scores/evaluate
func (h *Handler) evaluateScore(w http.ResponseWriter, r *http.Request) {
	var req domain.ScoringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SubmissionID == "" {
		writeError(w, http.StatusBadRequest, "submission_id is required")
		return
	}

	if req.ScoreType == "" {
		writeError(w, http.StatusBadRequest, "score_type is required")
		return
	}

	score, err := h.scoringService.Evaluate(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to evaluate score: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, score)
}

// getScore handles GET /api/v1/scores/{id}
func (h *Handler) getScore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "score id is required")
		return
	}

	score, err := h.scoringService.GetScore(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "score not found")
		return
	}

	writeJSON(w, http.StatusOK, score)
}

// getScoresBySubmission handles GET /api/v1/submissions/{submission_id}/scores
func (h *Handler) getScoresBySubmission(w http.ResponseWriter, r *http.Request) {
	submissionID := chi.URLParam(r, "submission_id")
	if submissionID == "" {
		writeError(w, http.StatusBadRequest, "submission_id is required")
		return
	}

	scores, err := h.scoringService.GetScoresBySubmission(r.Context(), submissionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve scores")
		return
	}

	writeJSON(w, http.StatusOK, scores)
}

// getEvaluationResult handles GET /api/v1/submissions/{submission_id}/result
func (h *Handler) getEvaluationResult(w http.ResponseWriter, r *http.Request) {
	submissionID := chi.URLParam(r, "submission_id")
	if submissionID == "" {
		writeError(w, http.StatusBadRequest, "submission_id is required")
		return
	}

	result, err := h.scoringService.GetEvaluationResult(r.Context(), submissionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "evaluation result not found: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// listScores handles GET /api/v1/scores
func (h *Handler) listScores(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	scores, err := h.scoringService.ListScores(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list scores")
		return
	}

	writeJSON(w, http.StatusOK, scores)
}

// deleteScore handles DELETE /api/v1/scores/{id}
func (h *Handler) deleteScore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "score id is required")
		return
	}

	if err := h.scoringService.DeleteScore(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "score not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// createRubric handles POST /api/v1/rubrics
func (h *Handler) createRubric(w http.ResponseWriter, r *http.Request) {
	var rubric domain.Rubric
	if err := json.NewDecoder(r.Body).Decode(&rubric); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.scoringService.CreateRubric(r.Context(), &rubric); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create rubric")
		return
	}

	writeJSON(w, http.StatusCreated, rubric)
}

// healthCheck handles GET /health
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) generateInterviewReport(w http.ResponseWriter, r *http.Request) {
	var req generateInterviewReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	// New honest scoring:
	//   1. Filter "real" answers (sender=user AND content >=12 chars).
	//      Two-letter "hi" / "ok" / blanks no longer count.
	//   2. If we have AI verdicts (preferred), score each answer from
	//      its verdict:  correct=90 / partial=55 / wrong=20 /
	//      skipped=0 / off_topic=10. This is the AI-graded path.
	//   3. Otherwise grade from content length only, capped at 70.
	//   4. Zero real answers → zeros + honest weakness, no canned
	//      "strengths" gift.
	correctness, clarity, completeness, relevance := 0.0, 0.0, 0.0, 0.0
	strengths := []string{}
	weaknesses := []string{}
	recommendations := []string{}

	type scoredAnswer struct {
		base    float64
		clarity float64
		verdict string
	}
	var scored []scoredAnswer
	var verdictCounts = map[string]int{}

	for _, msg := range req.Messages {
		if msg.Sender != "user" {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if utf8.RuneCountInString(content) < 12 {
			continue
		}

		var base, clr float64
		switch msg.Verdict {
		case "correct":
			base, clr = 90, 92
		case "partial":
			base, clr = 55, 58
		case "wrong":
			base, clr = 20, 30
		case "skipped":
			base, clr = 0, 0
		case "off_topic":
			base, clr = 10, 20
		default:
			// No AI verdict — fall back to length heuristic, cap 70.
			runes := utf8.RuneCountInString(content)
			base = math.Min(70, 25+float64(runes)/8)
			clr = math.Min(70, 30+float64(runes)/9)
		}
		scored = append(scored, scoredAnswer{base: base, clarity: clr, verdict: msg.Verdict})
		if msg.Verdict != "" {
			verdictCounts[msg.Verdict]++
		}
	}

	if len(scored) == 0 {
		// Zero substantive answers — refuse to fabricate a score.
		correctness, clarity, completeness, relevance = 0, 0, 0, 0
		weaknesses = append(weaknesses,
			"Не было ни одного развёрнутого ответа — оценивать нечего.")
		recommendations = append(recommendations,
			"Попробуйте пройти интервью заново и отвечайте на каждый вопрос хотя бы 1–2 предложениями.")
	} else {
		totalBase, totalClr := 0.0, 0.0
		for _, s := range scored {
			totalBase += s.base
			totalClr += s.clarity
		}
		n := float64(len(scored))
		correctness = round2(totalBase / n)
		clarity = round2(totalClr / n)
		// Completeness penalises 'skipped' / 'wrong' more.
		penalty := 0.0
		penalty += float64(verdictCounts["skipped"]) * 7
		penalty += float64(verdictCounts["wrong"]) * 5
		penalty += float64(verdictCounts["off_topic"]) * 3
		completeness = clampScore(correctness - penalty/n)
		// Relevance from off_topic ratio.
		offRatio := float64(verdictCounts["off_topic"]) / n
		relevance = clampScore(85 - offRatio*60)

		// Adaptive narrative from verdict mix.
		if verdictCounts["correct"] >= verdictCounts["wrong"]+verdictCounts["off_topic"]+verdictCounts["skipped"] {
			strengths = append(strengths, "Большая часть ответов точная и по делу.")
		}
		if verdictCounts["partial"] > 0 {
			weaknesses = append(weaknesses, "Часть ответов раскрыта частично — не хватает edge-cases и trade-offs.")
		}
		if verdictCounts["wrong"] > 0 {
			weaknesses = append(weaknesses, "Есть фактические ошибки в ответах — стоит подтянуть базу.")
		}
		if verdictCounts["skipped"] > 0 {
			weaknesses = append(weaknesses, "Несколько вопросов пропущено — заметные пробелы в темах.")
		}
		if verdictCounts["off_topic"] > 0 {
			weaknesses = append(weaknesses, "Иногда ответы уходили в сторону от вопроса.")
		}
		if len(strengths) == 0 {
			strengths = append(strengths, "Есть начальный технический контекст.")
		}
		recommendations = append(recommendations,
			"Тренировать структуру: тезис → аргументы → пример → trade-offs.",
			"Подкреплять ответы метриками (latency, p95, throughput).")
	}

	overall := round2((correctness + clarity + completeness + relevance) / 4)

	report := interviewReport{
		SessionID:       req.SessionID,
		Correctness:     correctness,
		Clarity:         clarity,
		Completeness:    completeness,
		Relevance:       relevance,
		OverallScore:    overall,
		Strengths:       strengths,
		Weaknesses:      weaknesses,
		Recommendations: recommendations,
		GeneratedAt:     time.Now(),
	}

	h.reportsMu.Lock()
	h.interviewReports[req.SessionID] = report
	h.reportsMu.Unlock()

	writeJSON(w, http.StatusCreated, report)
}

func (h *Handler) getInterviewReport(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	h.reportsMu.RLock()
	report, ok := h.interviewReports[sessionID]
	h.reportsMu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return round2(v)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
