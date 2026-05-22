-- Лог практики через Telegram-бота.
--
-- Каждый раз, когда юзер отвечает в боте и softskills-сервис возвращает
-- score, мы пишем сюда строку. Используется для:
--   • /report — недельная сводка по запросу
--   • weekly auto-push (вс 18:00 UTC) — без явного запроса
--   • Skill Map в Profile (когда подключим адаптивную сложность)
--
-- chat_id хранится отдельно от user_id потому, что бот общается с
-- юзером по chat_id напрямую и не всегда хочет JOIN'ить telegram_links
-- ради простого INSERT'а на горячем пути.

CREATE TABLE IF NOT EXISTS telegram_practice_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID,
    chat_id     BIGINT NOT NULL,
    topic       TEXT NOT NULL DEFAULT 'general',
    question    TEXT NOT NULL,
    answer      TEXT NOT NULL,
    score       REAL,                              -- 0..100, может быть NULL при оффлайн-fallback
    verdict     TEXT,                              -- correct / partial / wrong / skipped / off_topic
    feedback    TEXT,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tg_practice_chat_created
    ON telegram_practice_log (chat_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tg_practice_user_created
    ON telegram_practice_log (user_id, created_at DESC)
    WHERE user_id IS NOT NULL;
