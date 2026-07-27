-- +goose Up 
CREATE TABLE user_ratings (
  user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
  media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
  rating_value INTEGER NOT NULL CHECK (rating_value BETWEEN 1 AND 10),
  watched_date DATE,
  deleted_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, media_id)
);


CREATE INDEX idx_user_ratings_user_created 
  ON user_ratings(user_id, created_at DESC, media_id DESC) 
  WHERE deleted_at IS NULL;
CREATE INDEX idx_user_ratings_media_created 
  ON user_ratings(media_id, created_at DESC, user_id DESC) 
  WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS user_ratings;
