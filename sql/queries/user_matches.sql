-- name: UpsertUserMatch :execrows
INSERT INTO user_matches (
  target_user_id,
  other_user_id,
  score,
  shared_media_count
) VALUES ($1, $2, $3, $4)
ON CONFLICT (target_user_id, other_user_id)
DO UPDATE SET 
  score = EXCLUDED.score, 
  shared_media_count = EXCLUDED.shared_media_count,
  last_recalculated_at = NOW();

-- name: GetUserMatches :many
SELECT
  um.target_user_id,
  um.other_user_id,
  um.score,
  um.shared_media_count,
  um.last_recalculated_at,
  users.id AS user_id,
  users.name AS user_name,
  users.created_at AS user_created_at,
  users.updated_at AS user_updated_at,
  users.profile_picture_url AS user_profile_picture_url,
  users.about AS user_about
FROM user_matches AS um
INNER JOIN users
ON users.id = um.other_user_id
WHERE um.target_user_id = sqlc.arg('target_user_id')
  AND (
    sqlc.narg('cursor_score')::double precision IS NULL
    OR (
      um.score,
      um.shared_media_count,
      um.last_recalculated_at,
      um.other_user_id
    ) < (
      sqlc.narg('cursor_score')::double precision,
      sqlc.narg('cursor_shared_media_count')::integer,
      sqlc.narg('cursor_last_recalculated_at')::timestamptz,
      sqlc.narg('cursor_other_user_id')::uuid
    )
  )
ORDER BY
  um.score DESC,
  um.shared_media_count DESC,
  um.last_recalculated_at DESC,
  um.other_user_id DESC
LIMIT sqlc.arg('limit');


-- name: CheckMatchExists :one
SELECT EXISTS (
  SELECT 1
  FROM user_matches
  WHERE target_user_id = $1
    AND other_user_id = $2
);
