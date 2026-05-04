
-- +goose Up
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  profile_picture_url TEXT,
  about TEXT,
  email CITEXT UNIQUE NOT NULL,
  password_hash BYTEA NOT NULL,
  activated BOOLEAN NOT NULL DEFAULT FALSE,
  version INTEGER NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE IF EXISTS users;
