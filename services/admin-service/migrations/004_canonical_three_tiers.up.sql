-- Платформа поддерживает РОВНО три тарифа: trial, pro, platinum.
-- Старые имена (free/basic/starter/team/enterprise) вызывали рассинхрон
-- между фронтом, gin-валидацией и CHECK-констрейнтом — оплата падала
-- 400 при выборе platinum/trial. Эта миграция приводит данные и
-- констрейнт к каноническому набору из трёх тарифов.

-- 1. Снимаем старый констрейнт, чтобы можно было переписать значения.
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_tier_check;

-- 2. Маппинг существующих строк на три тарифа.
UPDATE subscriptions SET tier = 'trial'    WHERE tier IN ('free');
UPDATE subscriptions SET tier = 'pro'      WHERE tier IN ('basic', 'starter');
UPDATE subscriptions SET tier = 'platinum' WHERE tier IN ('team', 'enterprise');

-- 3. Новый дефолт и констрейнт.
ALTER TABLE subscriptions ALTER COLUMN tier SET DEFAULT 'trial';

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_tier_check
    CHECK (tier::text = ANY (ARRAY[
        'trial'::varchar,
        'pro'::varchar,
        'platinum'::varchar
    ]::text[]));
