
-- name: SearchUsers :many
SELECT id,
    name,
    email,
    created_at, 
    updated_at,
    profile_picture_url,
    about
FROM users 
WHERE name ILIKE '%' || sqlc.arg('query') || '%'
ORDER BY similarity(name, sqlc.arg('query')) DESC
LIMIT sqlc.arg('limit');

