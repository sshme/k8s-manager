-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type INTEGER NOT NULL DEFAULT 1, -- 1 = USER, 2 = SERVICE
    role INTEGER NOT NULL DEFAULT 0,  -- 0 = BASE, 1 = ADMIN
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on name for search
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);

-- Create index on created_at for sorting
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

