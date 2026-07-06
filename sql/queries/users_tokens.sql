-- name: GetTokenForUser :one

SELECT users.id, 
      users.created_at, 
      users.name, 
      users.email, 
      users.password_hash, 
      users.activated, 
      users.version, 
      users.about,
      users.profile_picture_url
FROM users 
INNER JOIN tokens
ON users.id = tokens.user_id
WHERE tokens.token_hash = $1
AND tokens.scope = $2
AND tokens.expiry > $3;
