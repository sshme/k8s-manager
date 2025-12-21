-- Create releases table
CREATE TABLE IF NOT EXISTS releases (
    id BIGSERIAL PRIMARY KEY,
    plugin_id BIGINT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    published_at TIMESTAMP NOT NULL DEFAULT NOW(),
    changelog TEXT,
    min_cli_version VARCHAR(50),
    min_k8s_version VARCHAR(50),
    is_latest BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, version)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_releases_plugin_id ON releases(plugin_id);
CREATE INDEX IF NOT EXISTS idx_releases_version ON releases(version);
CREATE INDEX IF NOT EXISTS idx_releases_published_at ON releases(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_releases_is_latest ON releases(plugin_id, is_latest) WHERE is_latest = TRUE;

