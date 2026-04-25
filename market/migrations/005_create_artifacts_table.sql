-- Create artifacts table
CREATE TABLE IF NOT EXISTS artifacts (
    id BIGSERIAL PRIMARY KEY,
    release_id BIGINT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    os VARCHAR(50) NOT NULL,
    arch VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'zip',
    size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL, -- SHA-256
    storage_path VARCHAR(512) NOT NULL,
    signature TEXT,
    key_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(release_id, os, arch)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_artifacts_release_id ON artifacts(release_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_platform ON artifacts(os, arch);
CREATE INDEX IF NOT EXISTS idx_artifacts_checksum ON artifacts(checksum);
