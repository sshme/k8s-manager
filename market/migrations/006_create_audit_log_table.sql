-- Create audit log table for tracking changes
CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL, -- 'plugin', 'release', 'artifact'
    entity_id BIGINT NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'create', 'update', 'delete', 'status_change'
    user_id VARCHAR(255),
    reason TEXT,
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at DESC);

