-- +goose Up

CREATE TABLE IF NOT EXISTS discovery_history (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
  next_eligible_at TIMESTAMP WITH TIME ZONE NOT NULL,
  PRIMARY KEY (user_id, media_id)
);

-- +goose Down

DROP TABLE IF EXISTS discovery_history;
