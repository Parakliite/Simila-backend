-- name: InsertMediaToWatchlist :one

INSERT INTO user_watchlist (
  user_id,
  media_id,
  source,
  source_match_id
) VALUES ($1, $2, $3, $4)
RETURNING user_id, media_id, source, source_match_id, created_at, status;

-- name: UpdateWatchlistItemStatus :one

UPDATE user_watchlist
SET status = $3
WHERE user_id = $1 AND media_id = $2
RETURNING user_id, media_id, source, source_match_id, created_at, status;

-- name: DeleteMediaFromWatchlist :execrows

DELETE FROM user_watchlist
WHERE user_id = $1 AND media_id = $2;

-- name: GetAllItemsInWatchlist :many

SELECT
  uw.user_id,
  uw.media_id,
  uw.source,
  uw.source_match_id,
  uw.created_at,
  uw.status,
  m.tmdb_id,
  m.title,
  m.original_title,
  m.poster_path,
  m.backdrop_path,
  m.overview,
  m.release_date,
  m.runtime,
  m.media_type
FROM user_watchlist AS uw
JOIN media AS m
ON uw.media_id = m.id
WHERE uw.user_id = $1
  AND (
    sqlc.arg('cursor_created_at')::timestamptz = '0001-01-01 00:00:00+00'::timestamptz
    OR (uw.created_at, uw.media_id) < (sqlc.arg('cursor_created_at'), sqlc.arg('cursor_media_id')::UUID)
  )
  AND (
    sqlc.narg('status')::status_type IS NULL OR uw.status = sqlc.narg('status')
  )
ORDER BY uw.created_at DESC, uw.media_id DESC
LIMIT $2;
