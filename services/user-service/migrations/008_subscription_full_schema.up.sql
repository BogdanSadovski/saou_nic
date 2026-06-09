-- =====================================================================
-- Полная схема subscriptions для admin-service биллинга.
--
-- Контекст: admin-service (config.yaml.dbname = user_service) пишет
-- подписки в user_service.subscriptions. Базовая таблица создаётся
-- в 003_create_billing_tables, но не содержит всех колонок, которые
-- ожидает Go-структура SubscriptionRepository. Эта миграция
-- добивает недостающие колонки.
--
-- Идемпотентна: ADD COLUMN IF NOT EXISTS — повторный прогон ничего
-- не сломает.
-- =====================================================================

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS auto_renew                BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS max_users                 INTEGER NOT NULL DEFAULT 1;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS max_interviews_per_month  INTEGER NOT NULL DEFAULT 100;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS max_storage_gb            INTEGER NOT NULL DEFAULT 5;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS max_concurrent_sessions   INTEGER NOT NULL DEFAULT 1;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS features                  JSONB   NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS metadata                  JSONB   NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS trial_ends_at             TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS canceled_at               TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS card_last4                VARCHAR(4);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS amount_cents              INTEGER;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS currency                  VARCHAR(3) DEFAULT 'BYN';
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS notes                     TEXT;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS stripe_subscription_id    VARCHAR(100);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS last_payment_at           TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS next_payment_at           TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS cancellation_reason       TEXT;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS paused_at                 TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS resumed_at                TIMESTAMPTZ;

-- Гарантируем, что updated_at точно есть (в 003 миграции его может не быть).
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
