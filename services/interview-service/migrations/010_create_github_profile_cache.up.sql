-- Кеш GitHub-профилей.
--
-- Сейчас fetchGitHubProfileAnalytics() ходит в GitHub API при каждом
-- открытии страницы — это медленно (5–10 сек), бьётся об rate-limit
-- неаутентифицированных запросов (60/час на IP) и теряет данные при
-- ребуте контейнера. Сохраняем полный JSON-payload в БД и проверяем
-- свежесть по `last_synced_at`. Re-fetch триггерится либо явно
-- (пользователь нажал «Обновить»), либо когда кеш старше TTL.
--
-- Поле raw_payload содержит весь githubImportResponse как jsonb,
-- чтобы при изменении схемы не пришлось писать новую миграцию —
-- читаем поля при чтении.

CREATE TABLE IF NOT EXISTS github_profile_cache (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL,
    github_username   TEXT NOT NULL,
    profile_url       TEXT NOT NULL,
    raw_payload       JSONB NOT NULL,
    last_synced_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, github_username)
);

CREATE INDEX IF NOT EXISTS idx_github_profile_cache_user
    ON github_profile_cache (user_id);

CREATE INDEX IF NOT EXISTS idx_github_profile_cache_synced
    ON github_profile_cache (last_synced_at);
