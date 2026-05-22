package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *Handler) RegisterRoutes() *mux.Router {
	router := mux.NewRouter()
	middleware := NewAuthMiddleware(h.authService)

	// API v1 prefix
	api := router.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)

	// Public auth routes
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", h.Register).Methods(http.MethodPost)
	auth.HandleFunc("/login", h.Login).Methods(http.MethodPost)
	auth.HandleFunc("/refresh", h.RefreshToken).Methods(http.MethodPost)

	// OAuth routes
	auth.HandleFunc("/oauth/{provider}", h.OAuthRedirect).Methods(http.MethodGet)
	auth.HandleFunc("/oauth/{provider}/callback", h.OAuthCallback).Methods(http.MethodGet)

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.RequireAuth)

	protected.HandleFunc("/users/me", h.GetProfile).Methods(http.MethodGet)
	protected.HandleFunc("/users/me", h.UpdateProfile).Methods(http.MethodPut)
	protected.HandleFunc("/users/me/password", h.ChangePassword).Methods(http.MethodPut)

	// Telegram-интеграция: фронт получает link-token, открывает
	// t.me/<bot>?start=<token>; сам бот биндит chat_id напрямую в БД.
	protected.HandleFunc("/integrations/telegram/link-token", h.IssueTelegramLinkToken).Methods(http.MethodPost)
	protected.HandleFunc("/integrations/telegram/status", h.GetTelegramStatus).Methods(http.MethodGet)
	protected.HandleFunc("/integrations/telegram", h.UnlinkTelegram).Methods(http.MethodDelete)

	// Admin routes
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.RequireRole("admin"))
	admin.HandleFunc("/users", h.ListUsers).Methods(http.MethodGet)
	admin.HandleFunc("/users/{id}", h.GetUser).Methods(http.MethodGet)

	return router
}
