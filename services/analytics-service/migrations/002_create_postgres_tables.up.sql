-- PostgreSQL tables for metadata storage (dashboards, exports, user sessions).

CREATE TABLE IF NOT EXISTS dashboards
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    tenant_id   VARCHAR(255) NOT NULL,
    description TEXT,
    widgets     JSONB NOT NULL DEFAULT '[]',
    created_by  VARCHAR(255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dashboards_tenant ON dashboards(tenant_id);

CREATE TABLE IF NOT EXISTS export_requests
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   VARCHAR(255) NOT NULL,
    format      VARCHAR(50) NOT NULL,
    filter      JSONB NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'pending',
    file_url    TEXT,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_export_requests_tenant ON export_requests(tenant_id);
CREATE INDEX idx_export_requests_status ON export_requests(status);

CREATE TABLE IF NOT EXISTS user_sessions
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(255),
    session_id  VARCHAR(255) NOT NULL UNIQUE,
    duration    FLOAT8 NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_session ON user_sessions(session_id);
CREATE INDEX idx_user_sessions_created ON user_sessions(created_at);

-- Funnels table for conversion funnel definitions.
CREATE TABLE IF NOT EXISTS funnels
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    tenant_id   VARCHAR(255) NOT NULL,
    steps       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_funnels_tenant ON funnels(tenant_id);
