-- name: UpsertUserReaction :one
WITH reaction AS (
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
  RETURNING id, reactor_user_id, rating_user_id, media_id, reaction, created_at, updated_at
)
SELECT
  reaction.id,
  reaction.reactor_user_id,
  reactor.name AS reactor_name,
  COALESCE(reactor.profile_picture_url, '') AS reactor_profile_picture_url,
  reaction.rating_user_id,
  rating_user.name AS rating_user_name,
  COALESCE(rating_user.profile_picture_url, '') AS rating_user_profile_picture_url,
  reaction.media_id,
  media.tmdb_id,
  media.title,
  media.original_title,
  media.poster_path,
  media.backdrop_path,
  media.media_type,
  user_ratings.rating_value,
  user_ratings.watched_date,
  user_ratings.created_at AS rating_created_at,
  user_ratings.updated_at AS rating_updated_at,
  reaction.reaction,
  reaction.created_at,
  reaction.updated_at
FROM reaction
JOIN users AS reactor ON reactor.id = reaction.reactor_user_id
JOIN users AS rating_user ON rating_user.id = reaction.rating_user_id
JOIN media ON media.id = reaction.media_id
JOIN user_ratings
  ON user_ratings.user_id = reaction.rating_user_id
 AND user_ratings.media_id = reaction.media_id;

-- name: DeleteUserReaction :execrows
DELETE FROM user_reactions
WHERE reactor_user_id = $1 AND rating_user_id = $2 AND media_id = $3;

-- name: GetUserReactionsForTargetUser :many
SELECT 
  user_reactions.id,
  user_reactions.reactor_user_id,
  reactor.name AS reactor_name,
  COALESCE(reactor.profile_picture_url, '') AS reactor_profile_picture_url,
  user_reactions.rating_user_id,
  rating_user.name AS rating_user_name,
  COALESCE(rating_user.profile_picture_url, '') AS rating_user_profile_picture_url,
  user_reactions.media_id,
  media.tmdb_id,
  media.title,
  media.original_title,
  media.poster_path,
  media.backdrop_path,
  media.media_type,
  user_ratings.rating_value,
  user_ratings.watched_date,
  user_ratings.created_at AS rating_created_at,
  user_ratings.updated_at AS rating_updated_at,
  user_reactions.reaction,
  user_reactions.created_at,
  user_reactions.updated_at
FROM user_reactions
JOIN users AS reactor ON reactor.id = user_reactions.reactor_user_id
JOIN users AS rating_user ON rating_user.id = user_reactions.rating_user_id
JOIN media ON media.id = user_reactions.media_id
JOIN user_ratings
  ON user_ratings.user_id = user_reactions.rating_user_id
 AND user_ratings.media_id = user_reactions.media_id
WHERE user_reactions.rating_user_id = $1
  AND (user_reactions.created_at, user_reactions.reactor_user_id) < (sqlc.arg('cursor_created_at'), sqlc.arg('cursor_reactor_user_id')::UUID)
ORDER BY user_reactions.created_at DESC, user_reactions.reactor_user_id DESC
LIMIT $2;

-- name: GetReactionCountForRating :one
SELECT COUNT(*) AS reaction_count
FROM user_reactions
WHERE rating_user_id = $1 AND media_id = $2;
