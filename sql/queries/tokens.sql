-- name: CreateToken :exec
INSERT INTO tokens (token_hash, user_id, expiry, scope)
VALUES ($1, $2, $3, $4);

-- name: DeleteAllTokensAllForUser :exec
DELETE FROM tokens
WHERE scope = $1 AND user_id = $2;
