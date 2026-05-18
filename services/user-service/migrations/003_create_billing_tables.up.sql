CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    renewal_date TIMESTAMPTZ,
    trial_end_date TIMESTAMPTZ,
    payment_method_id TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_active ON subscriptions(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_subscriptions_created_at ON subscriptions(created_at DESC);

CREATE TABLE IF NOT EXISTS payment_intents (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier VARCHAR(32) NOT NULL,
    billing_cycle VARCHAR(32) NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(8) NOT NULL,
    status VARCHAR(32) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    client_secret TEXT NOT NULL UNIQUE,
    payment_method_id TEXT,
    promo_code_id TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_intents_user_created ON payment_intents(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_intents_status ON payment_intents(status);

CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY,
    intent_id UUID REFERENCES payment_intents(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(8) NOT NULL,
    status VARCHAR(32) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    external_reference TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_user_created ON payment_transactions(user_id, created_at DESC);
