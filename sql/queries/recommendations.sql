
-- name: GetUserRecommendations :many

SELECT 
  m.id,
  m.tmdb_id,
  m.title,
  m.original_title,
  m.poster_path,
  m.overview,
  m.release_date,
  m.media_type
FROM user_ratings AS ur
JOIN media AS m ON m.id = ur.media_id
WHERE ur.user_id = sqlc.arg(target_user_id)
  AND ur.deleted_at IS NULL
  AND ur.rating_value >= sqlc.arg(threshold)
  AND NOT EXISTS (
    SELECT 1
    FROM user_ratings AS ur_b
    WHERE ur_b.media_id = m.id
    AND ur_b.user_id = sqlc.arg(user_id)
    AND ur_b.deleted_at IS NULL
  )
ORDER BY ur.rating_value DESC, ur.created_at DESC, m.id DESC
LIMIT sqlc.arg('limit');
