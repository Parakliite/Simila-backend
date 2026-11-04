-- name: GetOnboardingMedia :many
SELECT
  m.id,
  m.created_at,
  m.updated_at,
  m.tmdb_id,
  m.title,
  m.original_title,
  m.poster_path,
  m.backdrop_path,
  m.overview,
  m.release_date,
  m.runtime,
  m.popularity,
  m.media_type,
  COALESCE((
    SELECT json_agg(g.name)
    FROM genres AS g
    JOIN media_genres AS mg ON g.id = mg.genre_id
    WHERE mg.media_id = m.id
  ), '[]'::json)::text AS genres
FROM media AS m
WHERE m.popularity >= $1
  AND NOT EXISTS (
    SELECT 1
    FROM user_ratings AS ur
    WHERE ur.user_id = $2
      AND ur.media_id = m.id
      AND ur.deleted_at IS NULL
  )
ORDER BY RANDOM()
LIMIT $3;

-- name: MarkUserOnboarded :one
UPDATE users
SET onboarded = TRUE,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, created_at, updated_at, profile_picture_url,
          about, activated, onboarded, version;
