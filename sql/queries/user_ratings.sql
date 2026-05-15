-- name: UpsertUserRating :one

INSERT INTO user_ratings (
  user_id,
  media_id,
  rating_value,
  watched_date
) VALUES ($1, $2, $3, $4) 
ON CONFLICT (user_id, media_id)
DO UPDATE SET 
  rating_value = EXCLUDED.rating_value, 
  watched_date = EXCLUDED.watched_date,
  deleted_at = NULL,
  updated_at = NOW()
RETURNING user_id, media_id, rating_value, watched_date, created_at, updated_at;

-- name: SoftDeleteUserRating :execrows

UPDATE user_ratings
SET deleted_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND media_id = $2 AND deleted_at IS NULL;

-- name: GetUsersRatings :many

SELECT 
  u.user_id, 
  u.media_id, 
  u.rating_value, 
  u.watched_date, 
  u.created_at, 
  u.updated_at,
  m.title,
  m.overview,
  m.poster_path,
  m.release_date,
  m.media_type
FROM user_ratings AS u
JOIN media AS m
ON u.media_id = m.id
WHERE u.user_id = $1
  AND u.deleted_at IS NULL
  AND (u.created_at, u.media_id) < (sqlc.arg('cursor_created_at'), sqlc.arg('cursor_media_id')::UUID)
ORDER BY u.created_at DESC, u.media_id DESC
LIMIT $2;

-- name: GetSingleUserRating :one
SELECT 
  u.user_id, 
  u.media_id, 
  u.rating_value, 
  u.watched_date, 
  u.created_at, 
  u.updated_at,
  m.title,
  m.overview,
  m.poster_path,
  m.release_date,
  m.media_type
FROM user_ratings AS u
JOIN media AS m
ON u.media_id = m.id
WHERE u.user_id = $1 AND u.media_id = $2 AND u.deleted_at IS NULL;

-- name: GetAllUserRatings :many
SELECT
    user_ratings.user_id,
    user_ratings.rating_value,
    user_ratings.media_id
FROM user_ratings
JOIN users ON user_ratings.user_id = users.id
WHERE user_ratings.user_id = $1;

-- name: GetAllRatingsForSingleMedia :many
SELECT
  ur.user_id,
  ur.media_id,
  ur.rating_value,
  ur.watched_date,
  ur.created_at,
  ur.updated_at,
  m.title,
  m.overview,
  m.poster_path,
  m.release_date,
  m.tmdb_id,
  m.backdrop_path,
  m.runtime,
  m.media_type
FROM media AS m
JOIN user_ratings AS ur
ON m.id = ur.media_id
WHERE media_id = $1
  AND ur.deleted_at IS NULL
  AND (ur.created_at, ur.user_id) < (sqlc.arg('cursor_created_at'), sqlc.arg('cursor_user_id')::UUID)
ORDER BY ur.created_at DESC, ur.user_id DESC
LIMIT $2;

