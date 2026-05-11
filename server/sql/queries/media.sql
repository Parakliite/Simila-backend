-- name: GetMedia :one

SELECT  media.id,
        media.tmdb_id,
        media.title,
        media.original_title,
        media.poster_path,
        media.backdrop_path,
        media.overview,
        media.release_date,
        media.runtime,
        media.media_type,
        (
            SELECT json_agg(genres.name)
            FROM genres
            JOIN media_genres ON genres.id = media_genres.genre_id
            WHERE media_genres.media_id = media.id
        ) AS genres
FROM media
WHERE media.tmdb_id = $1 AND media.media_type = $2;


-- name: GetGenre :one
SELECT id
FROM genres
WHERE name = $1;

-- name: CreateMedia :one
INSERT INTO media (
  created_at, 
  updated_at, 
  tmdb_id, 
  title, 
  original_title, 
  poster_path, 
  backdrop_path, 
  overview, 
  release_date, 
  runtime, 
  media_type
) 
VALUES (
  NOW(), NOW(), $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: CreateMediaGenre :exec
INSERT INTO media_genres (
    created_at,
    genre_id,
    media_id
) VALUES (
    NOW(), $1, $2
);

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

