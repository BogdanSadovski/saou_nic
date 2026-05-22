package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Telegram-привязка: выдаёт одноразовый токен, который пользователь
// откроет как t.me/<bot>?start=<token>. Сам telegram-bot service
// читает таблицу telegram_links напрямую (см. services/telegram-bot/db.py).
//
// Здесь — только запись токена. Endpoint защищён RequireAuth.

type telegramLinkTokenResponse struct {
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expires_at"`
	BotUsername string    `json:"bot_username"`
	DeepLink    string    `json:"deep_link"`
}

type telegramStatusResponse struct {
	Linked        bool       `json:"linked"`
	ChatID        *int64     `json:"chat_id,omitempty"`
	TgUsername    *string    `json:"tg_username,omitempty"`
	NotificationsPaused bool `json:"notifications_paused"`
	DailyHourUTC  int        `json:"daily_push_hour_utc"`
	LinkedAt      *time.Time `json:"linked_at,omitempty"`
}

func generateLinkToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func botUsername() string {
	if v := os.Getenv("TG_BOT_USERNAME"); v != "" {
		return v
	}
	return "realsync_practice_bot"
}

// IssueTelegramLinkToken handles POST /api/v1/integrations/telegram/link-token.
func (h *Handler) IssueTelegramLinkToken(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	token, err := generateLinkToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	expiresAt := time.Now().UTC().Add(30 * time.Minute)

	if err := h.userService.UpsertTelegramLinkToken(r.Context(), userID, token, expiresAt); err != nil {
		log.Printf("telegram link-token persist failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "failed to persist token")
		return
	}

	bot := botUsername()
	respondWithJSON(w, http.StatusOK, telegramLinkTokenResponse{
		Token:       token,
		ExpiresAt:   expiresAt,
		BotUsername: bot,
		DeepLink:    fmt.Sprintf("https://t.me/%s?start=%s", bot, token),
	})
}

// GetTelegramStatus — GET /api/v1/integrations/telegram/status.
// Фронт зовёт перед/после привязки, чтобы понять есть ли активный chat_id.
func (h *Handler) GetTelegramStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, err := h.userService.GetTelegramStatus(r.Context(), userID)
	if err != nil {
		log.Printf("telegram status read failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "failed to read status")
		return
	}
	respondWithJSON(w, http.StatusOK, status)
}

// UnlinkTelegram — DELETE /api/v1/integrations/telegram. Снимает chat_id.
func (h *Handler) UnlinkTelegram(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.userService.UnlinkTelegram(r.Context(), userID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to unlink")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// helper, чтобы избежать пустого импорта json в файле без других hits.
var _ = json.Marshal
var _ = uuid.New
var _ = context.Background
