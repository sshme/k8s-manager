-- Create table with plugins installed by users
ALTER TABLE artifacts ALTER COLUMN type SET DEFAULT 'zip';

CREATE TABLE IF NOT EXISTS plugin_installations (
    user_id VARCHAR(255) NOT NULL,
    plugin_id BIGINT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    installed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, plugin_id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_installations_user_id ON plugin_installations(user_id);
CREATE INDEX IF NOT EXISTS idx_plugin_installations_plugin_id ON plugin_installations(plugin_id);
CREATE INDEX IF NOT EXISTS idx_plugin_installations_installed_at ON plugin_installations(installed_at DESC);
