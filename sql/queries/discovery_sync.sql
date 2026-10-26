-- name: GetDiscoverySyncState :one
SELECT media_type, next_page, consecutive_failures, paused_until, last_success_at
FROM discovery_sync_state
WHERE media_type = $1;

-- name: UpdateDiscoverySyncSuccess :exec
UPDATE discovery_sync_state
SET next_page = $2,
    consecutive_failures = 0,
    paused_until = NULL,
    last_success_at = NOW()
WHERE media_type = $1;

-- name: UpdateDiscoverySyncFailure :exec
UPDATE discovery_sync_state
SET consecutive_failures = $2,
    paused_until = $3
WHERE media_type = $1;
