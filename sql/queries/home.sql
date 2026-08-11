-- name: GetTopRatedByMatches :many
SELECT
  m.id            AS media_id,
  m.title         AS title,
  m.poster_path   AS poster_path,
  m.media_type    AS media_type,
  u.id            AS match_user_id,
  u.name          AS match_name,
  u.profile_picture_url AS match_avatar,
  um.score        AS match_score,
  ur.rating_value AS rating_value,
  ur.created_at   AS rated_at
FROM user_matches AS um
JOIN user_ratings AS ur
  ON ur.user_id = um.other_user_id
  AND ur.deleted_at IS NULL
  AND ur.created_at >= NOW() - INTERVAL '14 days'
  AND ur.rating_value >= 7
JOIN media AS m
  ON m.id = ur.media_id
JOIN users AS u
  ON u.id = um.other_user_id
WHERE um.target_user_id = $1
  AND ur.media_id NOT IN (
    SELECT media_id FROM user_ratings
    WHERE user_id = $1 AND deleted_at IS NULL
  )
ORDER BY ur.rating_value DESC, um.score DESC, ur.created_at DESC
LIMIT $2;
