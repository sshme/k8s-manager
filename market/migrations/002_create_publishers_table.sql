-- Create publishers table
CREATE TABLE IF NOT EXISTS publishers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    website_url VARCHAR(512),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on name for search
CREATE INDEX IF NOT EXISTS idx_publishers_name ON publishers(name);

