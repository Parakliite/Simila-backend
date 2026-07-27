-- +goose Up

CREATE TABLE reaction_comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reaction_id UUID NOT NULL REFERENCES user_reactions(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (LENGTH(TRIM(body)) BETWEEN 1 AND 1000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reaction_comments_reaction_created
ON reaction_comments (reaction_id, created_at DESC, id DESC);

-- +goose Down

DROP TABLE IF EXISTS reaction_comments;
