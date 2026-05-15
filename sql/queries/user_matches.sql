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
