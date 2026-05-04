-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password_hash,
    profile_picture_url,
    about
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, name, email, created_at, updated_at, profile_picture_url, about, activated, version;

-- name: GetUserByEmail :one
SELECT id, name, email, created_at, updated_at, profile_picture_url, password_hash, about, activated, version
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, name, email, created_at, updated_at, profile_picture_url, password_hash, about, activated, version
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET name = $1,
    email = $2,
    profile_picture_url = $3,
    about = $4,
    activated = $5,
    password_hash = $6,
    version = version + 1,
    updated_at = NOW()
WHERE id = $7 AND version = $8
RETURNING version;
