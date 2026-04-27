-- Migration: 001_create_events
-- Description: Creates the core events table in ClickHouse for analytics event storage.

CREATE TABLE IF NOT EXISTS events
(
    id            String,
    type          LowCardinality(String),
    user_id       String,
    session_id    String,
    tenant_id     LowCardinality(String),
    url           String,
    referrer      String,
    user_agent    String,
    ip            String,
    country       LowCardinality(String),
    city          LowCardinality(String),
    device        LowCardinality(String),
    os            LowCardinality(String),
    browser       LowCardinality(String),
    properties    String,
    timestamp     DateTime64(3),
    processed_at  DateTime64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (tenant_id, type, timestamp)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Aggregated metrics table for pre-computed analytics.
CREATE TABLE IF NOT EXISTS aggregated_metrics
(
    id                  String,
    tenant_id           LowCardinality(String),
    window_start        DateTime64(3),
    window_end          DateTime64(3),
    granularity         LowCardinality(String),
    total_events        Int64,
    unique_users        Int64,
    unique_sessions     Int64,
    page_views          Int64,
    clicks              Int64,
    conversions         Int64,
    avg_session_duration Float64,
    bounce_rate         Float64,
    errors              Int64,
    created_at          DateTime64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(window_start)
ORDER BY (tenant_id, granularity, window_start)
SETTINGS index_granularity = 8192;

-- Materialized view for automatic event counting.
CREATE TABLE IF NOT EXISTS event_counts
(
    tenant_id   LowCardinality(String),
    event_type  LowCardinality(String),
    event_date  Date,
    event_count SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_type, event_date);

-- Populate event_counts from events.
CREATE MATERIALIZED VIEW IF NOT EXISTS event_counts_mv
TO event_counts
AS
SELECT
    tenant_id,
    type as event_type,
    toDate(timestamp) as event_date,
    sumState(1) as event_count
FROM events
GROUP BY tenant_id, type, toDate(timestamp);
