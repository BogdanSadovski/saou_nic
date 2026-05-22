-- Дополнительное состояние Telegram-бота:
--   • streak (UNIX дата последнего ответа + длина серии)
--   • выданные «term of the day» / «tech history» — чтобы не повторять
--   • история pomodoro-сессий и system-design челленджей
--
-- Не объединяем с telegram_links — там горячая таблица для биндинга,
-- эту трогаем редко.

CREATE TABLE IF NOT EXISTS telegram_user_state (
    user_id              UUID PRIMARY KEY,
    chat_id              BIGINT,
    -- streak: серия подряд дней с ≥1 ответом
    streak_current       INTEGER NOT NULL DEFAULT 0,
    streak_best          INTEGER NOT NULL DEFAULT 0,
    streak_last_day      DATE,
    -- избегаем повторов в «term of the day»
    used_term_ids        INTEGER[] NOT NULL DEFAULT '{}',
    -- последний выпуск daily news (UTC date)
    last_news_date       DATE,
    last_term_date       DATE,
    last_history_date    DATE,
    -- активный pomodoro (epoch конца)
    pomodoro_until       TIMESTAMP WITH TIME ZONE,
    pomodoro_phase       TEXT,        -- 'focus' | 'break'
    updated_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tgstate_chat ON telegram_user_state (chat_id);
