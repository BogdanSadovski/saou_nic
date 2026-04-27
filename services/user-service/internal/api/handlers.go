package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/real-ass/user-service/internal/domain"
	"github.com/real-ass/user-service/internal/service"
)

type Handler struct {
	authService  *service.AuthService
	userService  *service.UserService
	oauthService *service.OAuthService
}

func NewHandler(authService *service.AuthService, userService *service.UserService, oauthService *service.OAuthService) *Handler {
	return &Handler{
		authService:  authService,
		userService:  userService,
		oauthService: oauthService,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "email, username, and password are required")
		return
	}

	createReq := domain.CreateUserRequest{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	}

	tokens, err := h.authService.Register(r.Context(), createReq)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			respondWithError(w, http.StatusConflict, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	respondWithJSON(w, http.StatusCreated, tokens)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			respondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	respondWithJSON(w, http.StatusOK, tokens)
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.authService.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	respondWithJSON(w, http.StatusOK, tokens)
}

func (h *Handler) OAuthRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]

	authURL := h.oauthService.GetAuthURL(provider)
	if authURL == "" {
		respondWithError(w, http.StatusBadRequest, "unsupported OAuth provider")
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]

	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "missing code parameter")
		return
	}

	tokens, err := h.oauthService.HandleCallback(r.Context(), provider, code)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to process OAuth callback")
		return
	}

	respondWithJSON(w, http.StatusOK, tokens)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userService.UpdateUser(r.Context(), userID, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid limit parameter")
			return
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if _, err := fmt.Sscanf(o, "%d", &offset); err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid offset parameter")
			return
		}
	}

	users, err := h.userService.ListUsers(r.Context(), limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	respondWithJSON(w, http.StatusOK, users)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, errorResponse{Error: message})
}
