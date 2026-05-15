-- =========================================
-- USERS
-- =========================================

-- name: CreateUser :execresult
INSERT INTO users (
  name, email, password_hash
) VALUES (
  $1, $2, $3
);

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: ShowSoftDeletedUsers :many
SELECT * FROM users
WHERE deleted_at IS NOT NULL;

-- name: SoftDeleteUser :exec
UPDATE users
SET deleted_at = NOW()
WHERE id = $1;

-- name: HardDeleteUser :exec
DELETE FROM users
WHERE id = $1;


-- =========================================
-- SESSIONS
-- =========================================

-- name: CreateSession :execresult
INSERT INTO sessions (
  user_id,
  token_hash,
  refresh_token_hash,
  expires_at,
  user_agent,
  ip_address
) VALUES (
  $1, $2, $3, $4, $5, $6
);

-- name: GetSessionByTokenHash :one
SELECT * FROM sessions
WHERE token_hash = $1
  AND deleted_at IS NULL
  AND expires_at > NOW();

-- name: UpdateSessionTokenHash :exec
UPDATE sessions
SET token_hash = $2
WHERE token_hash = $1
  AND deleted_at IS NULL
  AND expires_at > NOW();

-- name: GetSessionByRefreshTokenHash :one
SELECT * FROM sessions
WHERE refresh_token_hash = $1
  AND deleted_at IS NULL
  AND expires_at > NOW();

-- name: UpdateSessionRefreshTokenHash :exec
UPDATE sessions
SET refresh_token_hash = $2
WHERE refresh_token_hash = $1
  AND deleted_at IS NULL
  AND expires_at > NOW();

-- name: SoftDeleteSessionByTokenHash :exec
UPDATE sessions
SET deleted_at = NOW()
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < NOW();