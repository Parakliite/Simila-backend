-- +goose Up

CREATE TYPE status_type AS ENUM (
  'watched',
  'not_watched'
);

CREATE TYPE source_type AS ENUM (
  'self',
  'from_match'
);

CREATE TABLE user_watchlist (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status status_type DEFAULT 'not_watched',
  source source_type NOT NULL DEFAULT 'self',
  source_match_id UUID REFERENCES users(id) ON DELETE CASCADE,

  PRIMARY KEY (user_id, media_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_watchlist;
