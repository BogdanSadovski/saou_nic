-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    channel VARCHAR(50) NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'normal',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    subject VARCHAR(500),
    body TEXT NOT NULL,
    recipient VARCHAR(500) NOT NULL,
    metadata JSONB,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for querying notifications by user
CREATE INDEX idx_notifications_user_id ON notifications (user_id);

-- Index for filtering by status (for processing pending notifications)
CREATE INDEX idx_notifications_status ON notifications (status);

-- Index for sorting by creation date
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);

-- Composite index for user notification queries with pagination
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);

-- Index for type-based filtering
CREATE INDEX idx_notifications_type ON notifications (type);

-- Index for channel-based filtering
CREATE INDEX idx_notifications_channel ON notifications (channel);

-- Partial index for pending notifications to speed up processing queue
CREATE INDEX idx_notifications_pending ON notifications (created_at ASC) WHERE status = 'pending';

-- Add a trigger to automatically update the updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_notifications_updated_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
