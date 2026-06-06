-- Откат к расширенному набору тарифов (free/basic/starter/pro/team/enterprise).
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_tier_check;

ALTER TABLE subscriptions ALTER COLUMN tier SET DEFAULT 'free';

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_tier_check
    CHECK (tier::text = ANY (ARRAY[
        'free'::varchar,
        'basic'::varchar,
        'starter'::varchar,
        'pro'::varchar,
        'team'::varchar,
        'enterprise'::varchar
    ]::text[]));
