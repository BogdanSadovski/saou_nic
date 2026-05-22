package service

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	// pgx/stdlib регистрирует драйвер "pgx" для database/sql,
	// чтобы тут можно было открыть лёгкое второе соединение к user_service DB
	// без дублирования pgx-pool из основного UserRepository.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// sanitizeDSN убирает несовместимые с pgx query-параметры
// (главное — `schema=public`, наследие prisma-style DSN из docker-compose).
// pgx падает с FATAL "unrecognized configuration parameter «schema»",
// если оставить.
func sanitizeDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	q := u.Query()
	for _, key := range []string{"schema"} {
		q.Del(key)
	}
	u.RawQuery = q.Encode()
	cleaned := u.String()
	// На всякий случай — если protocol prefix пропал, восстановим.
	if !strings.HasPrefix(cleaned, "postgres") {
		cleaned = "postgresql://" + cleaned
	}
	return cleaned
}

// TelegramStatus — DTO для GET /integrations/telegram/status.
type TelegramStatus struct {
	Linked              bool       `json:"linked"`
	ChatID              *int64     `json:"chat_id,omitempty"`
	TgUsername          *string    `json:"tg_username,omitempty"`
	NotificationsPaused bool       `json:"notifications_paused"`
	DailyHourUTC        int        `json:"daily_push_hour_utc"`
	LinkedAt            *time.Time `json:"linked_at,omitempty"`
}

// UpsertTelegramLinkToken создаёт/обновляет токен для последующей
// привязки в telegram-bot service.
func (s *UserService) UpsertTelegramLinkToken(
	ctx context.Context,
	userID uuid.UUID,
	token string,
	expiresAt time.Time,
) error {
	db, err := s.userDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO telegram_links (user_id, link_token, link_token_expires_at, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET link_token = EXCLUDED.link_token,
		    link_token_expires_at = EXCLUDED.link_token_expires_at,
		    updated_at = NOW()
	`, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("upsert telegram token: %w", err)
	}
	return nil
}

// GetTelegramStatus читает текущее состояние привязки.
func (s *UserService) GetTelegramStatus(ctx context.Context, userID uuid.UUID) (*TelegramStatus, error) {
	db, err := s.userDB()
	if err != nil {
		return nil, err
	}
	var (
		chatID       sql.NullInt64
		username     sql.NullString
		paused       sql.NullBool
		hour         sql.NullInt32
		linkedAt     sql.NullTime
		hasRow       bool
	)
	row := db.QueryRowContext(ctx, `
		SELECT chat_id, tg_username, notifications_paused, daily_push_hour_utc, linked_at
		FROM telegram_links WHERE user_id = $1
	`, userID)
	err = row.Scan(&chatID, &username, &paused, &hour, &linkedAt)
	if err == sql.ErrNoRows {
		// нет записи — не привязан
		return &TelegramStatus{Linked: false, DailyHourUTC: 6}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read telegram status: %w", err)
	}
	hasRow = true
	_ = hasRow

	status := &TelegramStatus{
		Linked:              chatID.Valid,
		NotificationsPaused: paused.Bool,
		DailyHourUTC:        int(hour.Int32),
	}
	if chatID.Valid {
		v := chatID.Int64
		status.ChatID = &v
	}
	if username.Valid {
		v := username.String
		status.TgUsername = &v
	}
	if linkedAt.Valid {
		v := linkedAt.Time
		status.LinkedAt = &v
	}
	if !hour.Valid {
		status.DailyHourUTC = 6
	}
	return status, nil
}

// UnlinkTelegram сбрасывает chat_id (и tg_username, linked_at).
func (s *UserService) UnlinkTelegram(ctx context.Context, userID uuid.UUID) error {
	db, err := s.userDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		UPDATE telegram_links
		SET chat_id = NULL, tg_username = NULL, linked_at = NULL, updated_at = NOW()
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("unlink telegram: %w", err)
	}
	return nil
}

// userDB — получаем *sql.DB напрямую через DATABASE_URL.
// UserRepository прячет соединение внутри pgx-pool, поэтому для
// "лёгких" таблиц (telegram_links), которые не входят в основной
// домен User, открываем второе соединение через стандартный database/sql.
func (s *UserService) userDB() (*sql.DB, error) {
	if s.tgDB != nil {
		return s.tgDB, nil
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:postgres_secret@postgres:5432/user_service?sslmode=disable"
	}
	dsn = sanitizeDSN(dsn)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open user_service db: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	s.tgDB = db
	return db, nil
}
