-- +goose Up

ALTER TABLE media
  ADD COLUMN popularity DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (popularity >= 0),
  ADD COLUMN popularity_updated_at TIMESTAMP WITH TIME ZONE;

CREATE TABLE discovery_sync_state (
    media_type TEXT PRIMARY KEY CHECK (media_type IN ('movie', 'tv')),
    next_page INTEGER NOT NULL DEFAULT 1 CHECK (next_page > 0),
    consecutive_failures INTEGER NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    paused_until TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE
);

INSERT INTO discovery_sync_state (media_type)
VALUES ('movie'), ('tv');

CREATE INDEX media_popularity_idx
ON media (popularity)
WHERE popularity > 0;

-- +goose Down

DROP INDEX IF EXISTS media_popularity_idx;
DROP TABLE IF EXISTS discovery_sync_state;

ALTER TABLE media
  DROP COLUMN IF EXISTS popularity_updated_at,
  DROP COLUMN IF EXISTS popularity;
