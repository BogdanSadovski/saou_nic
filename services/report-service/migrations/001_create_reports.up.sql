-- Create reports table
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id VARCHAR(255) NOT NULL,
    interview_id VARCHAR(255),
    assessment_id VARCHAR(255),
    type VARCHAR(50) NOT NULL,
    format VARCHAR(10) NOT NULL DEFAULT 'pdf',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    title VARCHAR(500) NOT NULL,
    description TEXT,
    file_url TEXT,
    file_name VARCHAR(500),
    file_size BIGINT,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    generated_by VARCHAR(255) NOT NULL DEFAULT 'system'
);

-- Create indexes for common queries
CREATE INDEX idx_reports_candidate_id ON reports(candidate_id);
CREATE INDEX idx_reports_status ON reports(status);
CREATE INDEX idx_reports_type ON reports(type);
CREATE INDEX idx_reports_format ON reports(format);
CREATE INDEX idx_reports_created_at ON reports(created_at DESC);
CREATE INDEX idx_reports_expires_at ON reports(expires_at) WHERE expires_at IS NOT NULL;

-- Create index for compound queries
CREATE INDEX idx_reports_candidate_status ON reports(candidate_id, status);
CREATE INDEX idx_reports_status_created ON reports(status, created_at DESC);

-- Add check constraints
ALTER TABLE reports ADD CONSTRAINT chk_report_status
    CHECK (status IN ('pending', 'generating', 'completed', 'failed', 'expired'));

ALTER TABLE reports ADD CONSTRAINT chk_report_format
    CHECK (format IN ('pdf', 'docx'));

ALTER TABLE reports ADD CONSTRAINT chk_report_type
    CHECK (type IN ('interview_report', 'candidate_summary', 'assessment_report', 'comparative_analysis'));

-- Add function to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for auto-updating updated_at
CREATE TRIGGER update_reports_updated_at
    BEFORE UPDATE ON reports
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
