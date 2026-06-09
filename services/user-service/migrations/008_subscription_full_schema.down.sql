-- Откат добавленных колонок. Базовые колонки (id/user_id/tier/status/dates)
-- оставляем — они из 003_create_billing_tables.

ALTER TABLE subscriptions DROP COLUMN IF EXISTS auto_renew;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS max_users;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS max_interviews_per_month;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS max_storage_gb;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS max_concurrent_sessions;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS features;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS metadata;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS trial_ends_at;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS canceled_at;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS card_last4;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS amount_cents;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS currency;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS notes;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS stripe_subscription_id;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS last_payment_at;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS next_payment_at;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS cancellation_reason;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS paused_at;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS resumed_at;
