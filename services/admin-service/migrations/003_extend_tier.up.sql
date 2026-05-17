-- Allow the public tier names (starter, team) used by the frontend
-- TIER_CATALOG. The original constraint only knew the four backend
-- canonical names — when the user clicks 'Team' on the checkout page
-- it submits tier=team and the insert/update fails 400 with
-- 'subscriptions_tier_check'.

ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_tier_check;

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
