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
        media.popularity,
        media.media_type,
        COALESCE((
            SELECT json_agg(genres.name)
            FROM genres
            JOIN media_genres ON genres.id = media_genres.genre_id
            WHERE media_genres.media_id = media.id
        ), '[]'::json)::text AS genres
FROM media
WHERE media.tmdb_id = $1 AND media.media_type = $2;

-- name: GetOrCreateMediaID :one
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
  popularity,
  media_type
)
VALUES (
  NOW(),
  NOW(),
  $1,
  'Unknown',
  'Unknown',
  '',
  '',
  '',
  '1970-01-01',
  1,
  0,
  $2
)
ON CONFLICT (tmdb_id, media_type)
DO UPDATE SET tmdb_id = EXCLUDED.tmdb_id
RETURNING id;


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
  popularity,
  popularity_updated_at,
  media_type
) 
VALUES (
  NOW(), NOW(), $1, $2, $3, $4, $5, $6, $7, $8, $9,
  CASE WHEN $9 > 0 THEN NOW() ELSE NULL END,
  $10
)
RETURNING *;

-- name: UpdateMediaPopularity :exec
UPDATE media
SET popularity = $1,
    popularity_updated_at = NOW()
WHERE tmdb_id = $2 AND media_type = $3;

-- name: CreateMediaGenre :exec
INSERT INTO media_genres (
    created_at,
    genre_id,
    media_id
) VALUES (
    NOW(), $1, $2
);

-- name: GetAllMedia :many
SELECT id
FROM media;
