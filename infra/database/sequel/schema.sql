-- =========================================
-- USERS
-- =========================================
-- Stores application users.
-- Supports soft deletion and audit timestamps.
CREATE TABLE users (
  id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- unique user identifier

  name TEXT NOT NULL, -- display name
  email TEXT NOT NULL UNIQUE, -- unique login identifier (should be normalized in app layer)
  password_hash TEXT NOT NULL, -- hashed password (never store plaintext)

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- creation timestamp
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- last update timestamp (auto-updated via trigger)
  deleted_at TIMESTAMPTZ -- soft delete marker (NULL = active)
);

-- =========================================
-- SESSIONS
-- =========================================
-- Represents a login session (one device / client instance).
-- Handles both access + refresh token lifecycle.
CREATE TABLE sessions (
  id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- unique session identifier

  user_id INTEGER NOT NULL
    REFERENCES users(id) ON DELETE CASCADE, -- owning user (cascade delete on user removal)

  token_hash TEXT NOT NULL UNIQUE, -- hash of access/session token (never store raw token)
  refresh_token_hash TEXT UNIQUE, -- hash of refresh token (nullable for non-refresh flows)

  expires_at TIMESTAMPTZ NOT NULL, -- hard expiration timestamp for the session

  user_agent TEXT, -- client metadata (browser/app)
  ip_address INET, -- client IP address (validated by Postgres)

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- creation timestamp
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- last update timestamp (auto-updated via trigger)
  deleted_at TIMESTAMPTZ -- soft delete marker (NULL = active / valid session)
);

-- =========================================
-- INDEXES
-- =========================================

-- Enforces uniqueness of refresh tokens when present.
-- Allows multiple NULL values (sessions without refresh tokens).
CREATE UNIQUE INDEX idx_refresh_token_unique
ON sessions(refresh_token_hash)
WHERE refresh_token_hash IS NOT NULL;

-- (Recommended) Speeds up queries for active sessions per user.
-- Example: "get all active sessions for a user"
CREATE INDEX idx_sessions_user_active
ON sessions(user_id)
WHERE deleted_at IS NULL;

-- =========================================
-- TRIGGERS
-- =========================================

-- Function to automatically update `updated_at` on row modification.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply auto-update trigger to users table
CREATE TRIGGER trg_users_updated
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Apply auto-update trigger to sessions table
CREATE TRIGGER trg_sessions_updated
BEFORE UPDATE ON sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
