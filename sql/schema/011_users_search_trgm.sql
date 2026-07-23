-- +goose Up

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_users_name_trgm
ON users USING GIN (name gin_trgm_ops);

-- +goose Down

DROP INDEX IF EXISTS idx_users_name_trgm;

DROP EXTENSION IF EXISTS pg_trgm;
