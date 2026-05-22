-- Telegram-интеграция.
--
-- Хранит привязку аккаунта RealSync к Telegram-чату.
--
--   link_token        — короткоживущий токен, сгенерированный фронтом.
--                       Юзер открывает t.me/<bot>?start=<link_token>;
--                       бот видит token в /start, ищет здесь, биндит chat_id.
--   link_token_expires_at — TTL ~30 минут на одноразовый токен.
--   chat_id           — Telegram chat_id (берётся из update.message.chat.id).
--                       При успешной привязке link_token занулится, а
--                       linked_at проставится в NOW().
--   notifications_paused — если true, бот не шлёт daily push.
--   daily_push_hour_utc — час (0..23) UTC, когда отправлять daily push.
--                       Дефолт 6 UTC = 09:00 MSK / 09:00 минск.
--   last_pushed_at    — чтобы не отправлять дважды в день при рестарте cron.

CREATE TABLE IF NOT EXISTS telegram_links (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  UUID NOT NULL UNIQUE,
    link_token               TEXT,
    link_token_expires_at    TIMESTAMP WITH TIME ZONE,
    chat_id                  BIGINT,
    tg_username              TEXT,
    notifications_paused     BOOLEAN NOT NULL DEFAULT FALSE,
    daily_push_hour_utc      SMALLINT NOT NULL DEFAULT 6,
    last_pushed_at           TIMESTAMP WITH TIME ZONE,
    linked_at                TIMESTAMP WITH TIME ZONE,
    created_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_daily_hour CHECK (daily_push_hour_utc BETWEEN 0 AND 23)
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_tg_link_token
    ON telegram_links (link_token)
    WHERE link_token IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tg_chat_id
    ON telegram_links (chat_id)
    WHERE chat_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tg_push_due
    ON telegram_links (daily_push_hour_utc)
    WHERE chat_id IS NOT NULL AND NOT notifications_paused;
