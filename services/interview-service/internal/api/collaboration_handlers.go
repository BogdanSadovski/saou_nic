package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/interview-platform/interview-service/internal/domain"
)

// AddCollaborator adds an interviewer to the session
func (h *Handler) AddCollaborator(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	var req domain.AddCollaboratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	// Verify user exists (optional - can be checked via user service)
	// TODO: Call user-service to verify user exists

	collab, err := h.repo.AddCollaborator(r.Context(), sessionID, req.UserID, req.Role)
	if err != nil {
		h.logger.WithError(err).Error("failed to add collaborator")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add collaborator"})
		return
	}

	// Broadcast via WebSocket to other collaborators
	h.broadcastCollaboratorJoined(sessionID, collab)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(collab)
}

// ListCollaborators returns active collaborators for session
func (h *Handler) ListCollaborators(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	collaborators, err := h.repo.ListCollaborators(r.Context(), sessionID)
	if err != nil {
		h.logger.WithError(err).Error("failed to list collaborators")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch collaborators"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"collaborators": collaborators,
		"count":         len(collaborators),
	})
}

// AddNote adds a collaboration note
func (h *Handler) AddNote(w http.ResponseWriter, r *http.Request) {
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

	var req domain.AddNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	note := &domain.CollaborationNote{
		SessionID: sessionID,
		AuthorID:  userID,
		Content:   req.Content,
		IsPinned:  req.IsPinned,
		Mentions:  req.Mentions,
	}

	note, err = h.repo.AddNote(r.Context(), note)
	if err != nil {
		h.logger.WithError(err).Error("failed to add note")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save note"})
		return
	}

	// Broadcast via WebSocket
	h.broadcastNote(sessionID, note)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

// GetNotes returns collaboration notes for session
func (h *Handler) GetNotes(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	// Pagination
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	notes, err := h.repo.ListNotes(r.Context(), sessionID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("failed to get notes")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch notes"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notes":  notes,
		"count":  len(notes),
		"limit":  limit,
		"offset": offset,
	})
}

// SubmitScore submits interviewer's score
func (h *Handler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	interviewerID := getUserIDFromContext(r.Context())
	if interviewerID == uuid.Nil {
		h.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req domain.SubmitScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	score := &domain.InterviewerScore{
		SessionID:           sessionID,
		InterviewerID:       interviewerID,
		TechnicalScore:      req.TechnicalScore,
		CommunicationScore:  req.CommunicationScore,
		ProblemSolvingScore: req.ProblemSolvingScore,
		CultureFitScore:     req.CultureFitScore,
		CodingQualityScore:  req.CodingQualityScore,
		Recommendation:      &req.Recommendation,
		Strengths:           &req.Strengths,
		AreasForImprovement: &req.AreasForImprovement,
		AdditionalComments:  &req.AdditionalComments,
	}

	score, err = h.repo.SubmitScore(r.Context(), score)
	if err != nil {
		h.logger.WithError(err).Error("failed to submit score")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save score"})
		return
	}

	// Calculate consensus after each score submission
	go func() {
		_, _ = h.repo.CalculateConsensus(r.Context(), sessionID)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(score)
}

// GetScores returns all scores for session
func (h *Handler) GetScores(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	scores, err := h.repo.GetScores(r.Context(), sessionID)
	if err != nil {
		h.logger.WithError(err).Error("failed to get scores")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch scores"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scores": scores,
		"count":  len(scores),
	})
}

// GetConsensus returns consensus score for session
func (h *Handler) GetConsensus(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := mux.Vars(r)["sessionId"]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	consensus, err := h.repo.GetConsensus(r.Context(), sessionID)
	if err != nil {
		h.logger.WithError(err).Error("failed to get consensus")
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch consensus"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(consensus)
}

// Helper functions for WebSocket broadcasting

func (h *Handler) broadcastCollaboratorJoined(sessionID uuid.UUID, collab *domain.InterviewCollaborator) {
	h.moduleMu.RLock()
	clients := h.moduleWS[sessionID]
	h.moduleMu.RUnlock()

	if clients == nil {
		return
	}

	message := map[string]interface{}{
		"type":         "collaborator_joined",
		"collaborator": collab,
	}

	h.broadcastToClients(clients, message)
}

func (h *Handler) broadcastNote(sessionID uuid.UUID, note *domain.CollaborationNote) {
	h.moduleMu.RLock()
	clients := h.moduleWS[sessionID]
	h.moduleMu.RUnlock()

	if clients == nil {
		return
	}

	message := map[string]interface{}{
		"type": "note_added",
		"note": note,
	}

	h.broadcastToClients(clients, message)
}

func (h *Handler) broadcastToClients(clients map[*websocket.Conn]struct{}, message interface{}) {
	data, _ := json.Marshal(message)

	for conn := range clients {
		go func(c *websocket.Conn) {
			c.WriteMessage(websocket.TextMessage, data)
		}(conn)
	}
}

// Register collaboration routes
func (h *Handler) RegisterCollaborationRoutes(router *mux.Router) {
	sessionRouter := router.PathPrefix("/interviews/sessions/{sessionId}").Subrouter()

	// Collaborators
	sessionRouter.HandleFunc("/collaborators", h.AddCollaborator).Methods("POST")
	sessionRouter.HandleFunc("/collaborators", h.ListCollaborators).Methods("GET")

	// Notes
	sessionRouter.HandleFunc("/notes", h.AddNote).Methods("POST")
	sessionRouter.HandleFunc("/notes", h.GetNotes).Methods("GET")

	// Scoring
	sessionRouter.HandleFunc("/score", h.SubmitScore).Methods("POST")
	sessionRouter.HandleFunc("/scores", h.GetScores).Methods("GET")
	sessionRouter.HandleFunc("/consensus", h.GetConsensus).Methods("GET")
}
