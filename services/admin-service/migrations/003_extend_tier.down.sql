ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_tier_check;
ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_tier_check
    CHECK (tier::text = ANY (ARRAY[
        'free'::varchar,
        'basic'::varchar,
        'pro'::varchar,
        'enterprise'::varchar
    ]::text[]));
