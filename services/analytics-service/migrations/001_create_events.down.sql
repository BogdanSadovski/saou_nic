-- Migration: 001_create_events (down)
-- Description: Drops the tables created by the up migration.

DROP TABLE IF EXISTS event_counts_mv;
DROP TABLE IF EXISTS event_counts;
DROP TABLE IF EXISTS aggregated_metrics;
DROP TABLE IF EXISTS events;
