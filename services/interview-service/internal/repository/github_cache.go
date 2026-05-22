package repository

// GitHub profile cache — обёртка над таблицей github_profile_cache,
// созданной миграцией 010. Храним сырой JSON и время последней
// синхронизации, чтобы handler мог принять решение «отдать кеш или
// сходить в GitHub API».
//
// Намеренно не парсим JSON в типизированный объект: схема ответа
// `fetchGitHubProfileAnalytics` богатая и часто меняется. Marshal в
// jsonb на запись, Unmarshal в указанный target — на чтение.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GitHubProfileCacheEntry — что отдаёт репозиторий поверх таблицы.
type GitHubProfileCacheEntry struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	GitHubUsername string
	ProfileURL     string
	RawPayload     []byte
	LastSyncedAt   time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Decode распаковывает RawPayload в произвольный target (обычно
// *githubImportResponse в API-слое).
func (e *GitHubProfileCacheEntry) Decode(target any) error {
	if len(e.RawPayload) == 0 {
		return errors.New("github_profile_cache: empty raw_payload")
	}
	return json.Unmarshal(e.RawPayload, target)
}

// GitHubProfileCache — методы добавляются к PostgresRepository.

// UpsertGitHubProfile сохраняет или обновляет кеш. Уникальный ключ —
// (user_id, github_username), поэтому при повторном импорте того же
// аккаунта старая запись перезаписывается.
func (r *PostgresRepository) UpsertGitHubProfile(
	ctx context.Context,
	userID uuid.UUID,
	username string,
	profileURL string,
	payload any,
) (*GitHubProfileCacheEntry, error) {
	rawJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal github payload: %w", err)
	}

	const q = `
		INSERT INTO github_profile_cache (user_id, github_username, profile_url, raw_payload, last_synced_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, NOW(), NOW())
		ON CONFLICT (user_id, github_username) DO UPDATE
		SET profile_url    = EXCLUDED.profile_url,
		    raw_payload    = EXCLUDED.raw_payload,
		    last_synced_at = NOW(),
		    updated_at     = NOW()
		RETURNING id, user_id, github_username, profile_url, raw_payload,
		          last_synced_at, created_at, updated_at`

	var e GitHubProfileCacheEntry
	row := r.db.QueryRowContext(ctx, q, userID, username, profileURL, string(rawJSON))
	if err := row.Scan(
		&e.ID, &e.UserID, &e.GitHubUsername, &e.ProfileURL,
		&e.RawPayload, &e.LastSyncedAt, &e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("upsert github profile: %w", err)
	}
	return &e, nil
}

// GetGitHubProfileForUser возвращает последний (по last_synced_at)
// сохранённый профиль пользователя. Используется на страницах, где
// нужно показать GitHub-блок без re-fetch'а.
func (r *PostgresRepository) GetGitHubProfileForUser(
	ctx context.Context,
	userID uuid.UUID,
) (*GitHubProfileCacheEntry, error) {
	const q = `
		SELECT id, user_id, github_username, profile_url, raw_payload,
		       last_synced_at, created_at, updated_at
		FROM github_profile_cache
		WHERE user_id = $1
		ORDER BY last_synced_at DESC
		LIMIT 1`

	var e GitHubProfileCacheEntry
	row := r.db.QueryRowContext(ctx, q, userID)
	if err := row.Scan(
		&e.ID, &e.UserID, &e.GitHubUsername, &e.ProfileURL,
		&e.RawPayload, &e.LastSyncedAt, &e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // нет кеша — сигнал «нужно фетчить»
		}
		return nil, fmt.Errorf("get github profile: %w", err)
	}
	return &e, nil
}

// GetGitHubProfileByUsername — вариация для случая, когда указан
// конкретный username (важно если у пользователя в кеше несколько
// аккаунтов).
func (r *PostgresRepository) GetGitHubProfileByUsername(
	ctx context.Context,
	userID uuid.UUID,
	username string,
) (*GitHubProfileCacheEntry, error) {
	const q = `
		SELECT id, user_id, github_username, profile_url, raw_payload,
		       last_synced_at, created_at, updated_at
		FROM github_profile_cache
		WHERE user_id = $1 AND github_username = $2`

	var e GitHubProfileCacheEntry
	row := r.db.QueryRowContext(ctx, q, userID, username)
	if err := row.Scan(
		&e.ID, &e.UserID, &e.GitHubUsername, &e.ProfileURL,
		&e.RawPayload, &e.LastSyncedAt, &e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get github profile by username: %w", err)
	}
	return &e, nil
}
