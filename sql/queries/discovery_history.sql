
-- name: GetDiscoveryHistoryExpiry :one
SELECT next_eligible_at
FROM discovery_history
WHERE user_id = $1 AND media_id = $2;

-- name: SetDiscoveryHistoryExpiry :exec
INSERT INTO discovery_history (user_id, media_id, next_eligible_at) 
VALUES ($1, $2, $3)
ON CONFLICT (user_id, media_id)
DO UPDATE SET
  next_eligible_at = EXCLUDED.next_eligible_at;

-- name: DeleteDiscoveryHistory :execrows
DELETE FROM discovery_history
WHERE user_id = $1 AND media_id = $2;
