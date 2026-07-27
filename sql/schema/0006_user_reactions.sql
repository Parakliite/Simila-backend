-- +goose Up

CREATE TYPE reaction_type AS ENUM (
  'watched_because_of_you',
  'great_pick',
  'curious',
  'hot_take'
);

CREATE TABLE user_reactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reactor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rating_user_id UUID NOT NULL,
  media_id UUID NOT NULL,
  reaction reaction_type NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (reactor_user_id, rating_user_id, media_id),

  FOREIGN KEY (rating_user_id, media_id)
    REFERENCES user_ratings(user_id, media_id)
    ON DELETE CASCADE
);

CREATE INDEX idx_user_reactions_rating_created
ON user_reactions (rating_user_id, created_at DESC, reactor_user_id DESC);

CREATE INDEX idx_user_reactions_rating_media
ON user_reactions (rating_user_id, media_id);

-- +goose Down

DROP INDEX IF EXISTS idx_user_reactions_rating_media;
DROP INDEX IF EXISTS idx_user_reactions_rating_created;
DROP TABLE IF EXISTS user_reactions;
DROP TYPE IF EXISTS reaction_type;
