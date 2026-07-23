-- +goose Up
CREATE TABLE user_matches (
  target_user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
  other_user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
  score FLOAT NOT NULL CHECK (score >= -1 AND score <= 1),
  shared_media_count INTEGER NOT NULL CHECK (shared_media_count >= 0),
  last_recalculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (target_user_id, other_user_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_matches;
