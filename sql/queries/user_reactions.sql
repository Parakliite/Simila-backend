-- name: UpsertUserReaction :one
INSERT INTO user_reactions (
  reactor_user_id,
  rating_user_id,
  media_id,
  reaction
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (reactor_user_id, rating_user_id, media_id)
DO UPDATE SET
  reaction = EXCLUDED.reaction,
  updated_at = NOW()
RETURNING id, reactor_user_id, rating_user_id, media_id, reaction, created_at, updated_at;

-- name: DeleteUserReaction :execrows
DELETE FROM user_reactions
WHERE reactor_user_id = $1 AND rating_user_id = $2 AND media_id = $3;

-- name: GetUserReactionsForTargetUser :many
SELECT 
  id,
  reactor_user_id,
  rating_user_id,
  media_id,
  reaction,
  created_at,
  updated_at
FROM user_reactions
WHERE rating_user_id = $1
  AND (created_at, reactor_user_id) < (sqlc.arg('cursor_created_at'), sqlc.arg('cursor_reactor_user_id')::UUID)
ORDER BY created_at DESC, reactor_user_id DESC
LIMIT $2;

-- name: GetReactionCountForRating :one
SELECT COUNT(*) AS reaction_count
FROM user_reactions
WHERE rating_user_id = $1 AND media_id = $2;
