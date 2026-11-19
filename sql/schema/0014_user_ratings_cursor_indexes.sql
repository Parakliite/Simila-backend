-- +goose Up
CREATE INDEX user_ratings_user_updated_idx
ON user_ratings (user_id, updated_at DESC, media_id DESC)
WHERE deleted_at IS NULL;

CREATE INDEX user_ratings_user_rating_updated_idx
ON user_ratings (user_id, rating_value DESC, updated_at DESC, media_id DESC)
WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS user_ratings_user_rating_updated_idx;
DROP INDEX IF EXISTS user_ratings_user_updated_idx;
