-- =====================================================================
-- Аудит-лог админ-действий.
--
-- Контекст: admin-service (GetDashboardStats / AuditLogRepository) ищет
-- эту таблицу в БД user_service (dbname=user_service в его config.yaml).
-- Без неё /api/admin/dashboard/stats отвечает 500.
--
-- Идемпотентна.
-- =====================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id        UUID,
    admin_email     VARCHAR(255),
    action          VARCHAR(100) NOT NULL,
    -- Старая пара (target_type, target_id) — оставляем для обратной
    -- совместимости со старым кодом, новый код пишет в resource_*.
    target_type     VARCHAR(50),
    target_id       UUID,
    -- Каноничные имена, которые ожидает Go-структура AuditLog.
    resource_type   VARCHAR(50),
    resource_id     UUID,
    reason          TEXT,
    status          VARCHAR(20) DEFAULT 'success',
    error_message   TEXT,
    details         JSONB DEFAULT '{}'::jsonb,
    metadata        JSONB DEFAULT '{}'::jsonb,
    request_id      VARCHAR(100),
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_admin    ON audit_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action   ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created  ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_status   ON audit_logs(status);
