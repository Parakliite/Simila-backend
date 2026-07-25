-- name: CreateToken :exec
INSERT INTO tokens (token_hash, user_id, expiry, scope)
VALUES ($1, $2, $3, $4);

-- name: GetUserFromToken :one
SELECT user_id, revoked_at
FROM tokens
WHERE token_hash = $1 AND scope = $2;

-- name: RevokeAllPreviousTokens :exec
UPDATE tokens
SET revoked_at = NOW()
WHERE user_id = $1 AND scope = $2;

-- name: RevokePreviousToken :execrows
UPDATE tokens
SET revoked_at = NOW()
WHERE token_hash = $1 AND scope = $2;

-- name: DeleteAllTokensAllForUser :exec
DELETE FROM tokens
WHERE scope = $1 AND user_id = $2;
