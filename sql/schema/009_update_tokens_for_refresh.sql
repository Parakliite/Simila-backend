
-- +goose Up
ALTER TABLE tokens
  RENAME COLUMN hash TO token_hash;

ALTER TABLE tokens
  ADD COLUMN created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
  ADD COLUMN revoked_at timestamp(0) with time zone;

CREATE INDEX IF NOT EXISTS tokens_user_id_scope_idx
  ON tokens (user_id, scope);

CREATE INDEX IF NOT EXISTS tokens_scope_expiry_revoked_idx
  ON tokens (scope, expiry, revoked_at);

-- +goose Down
DROP INDEX IF EXISTS tokens_scope_expiry_revoked_idx;
DROP INDEX IF EXISTS tokens_user_id_scope_idx;

ALTER TABLE tokens
  DROP COLUMN revoked_at,
  DROP COLUMN created_at;

ALTER TABLE tokens
  RENAME COLUMN token_hash TO hash;
