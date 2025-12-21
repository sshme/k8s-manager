-- Create plugins table
CREATE TABLE IF NOT EXISTS plugins (
    id BIGSERIAL PRIMARY KEY,
    identifier VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    publisher_id BIGINT NOT NULL REFERENCES publishers(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'hidden', 'blocked')),
    trust_status VARCHAR(20) NOT NULL DEFAULT 'community' CHECK (trust_status IN ('official', 'verified', 'community')),
    source_url VARCHAR(512),
    docs_url VARCHAR(512),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_plugins_identifier ON plugins(identifier);
CREATE INDEX IF NOT EXISTS idx_plugins_name ON plugins(name);
CREATE INDEX IF NOT EXISTS idx_plugins_category ON plugins(category);
CREATE INDEX IF NOT EXISTS idx_plugins_publisher_id ON plugins(publisher_id);
CREATE INDEX IF NOT EXISTS idx_plugins_status ON plugins(status);
CREATE INDEX IF NOT EXISTS idx_plugins_trust_status ON plugins(trust_status);

