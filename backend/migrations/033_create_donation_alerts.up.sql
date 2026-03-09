CREATE TABLE donation_alerts (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id  VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    donation_id VARCHAR(36) REFERENCES donations(id) ON DELETE SET NULL,
    donor_type  VARCHAR(10) NOT NULL,
    donor_id    VARCHAR(36) NOT NULL,
    alert_type  VARCHAR(30) NOT NULL,
    severity    VARCHAR(10) NOT NULL DEFAULT 'warning',
    status      VARCHAR(15) NOT NULL DEFAULT 'new',
    details     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(36) REFERENCES users(id)
);

CREATE INDEX idx_donation_alerts_status ON donation_alerts (status);
CREATE INDEX idx_donation_alerts_project ON donation_alerts (project_id);
CREATE INDEX idx_donation_alerts_created ON donation_alerts (created_at DESC);
CREATE INDEX idx_donation_alerts_dedup ON donation_alerts (project_id, donor_type, donor_id, alert_type, created_at DESC);
