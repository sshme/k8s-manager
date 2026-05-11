-- Ensure a plugin has at most one latest release.
CREATE UNIQUE INDEX IF NOT EXISTS idx_releases_one_latest_per_plugin
ON releases(plugin_id)
WHERE is_latest = TRUE;

-- Ensure artifact checksums are valid SHA-256 hex strings.
ALTER TABLE artifacts
ADD CONSTRAINT artifacts_checksum_sha256
CHECK (checksum ~ '^[a-f0-9]{64}$')
NOT VALID;

