-- name: CreateToken :exec
INSERT INTO tokens (token_hash, session_id, user_id, expiry, scope)
VALUES ($1, $2, $3, $4, $5);

-- name: GetUserFromToken :one
SELECT user_id, session_id
FROM tokens
WHERE token_hash = $1
  AND scope = $2
  AND revoked_at IS NULL
  AND expiry > NOW();

-- name: RevokeAllTokensForSession :exec 
UPDATE tokens
SET revoked_at = NOW()
WHERE session_id = $1 AND scope = $2 AND revoked_at IS NULL;

-- name: RevokeAllPreviousTokens :exec
UPDATE tokens
SET revoked_at = NOW()
WHERE user_id = $1 AND scope = $2 AND revoked_at IS NULL;

-- name: RevokePreviousToken :execrows
UPDATE tokens
SET revoked_at = NOW()
WHERE token_hash = $1 AND scope = $2 AND revoked_at IS NULL;

-- name: DeleteAllTokensAllForUser :exec
DELETE FROM tokens
WHERE scope = $1 AND user_id = $2;
