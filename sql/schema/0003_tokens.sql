
-- +goose Up
CREATE TABLE IF NOT EXISTS tokens (
  token_hash bytea PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
  session_id UUID,
  created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
  expiry timestamp(0) with time zone NOT NULL,
  scope text NOT NULL,
  revoked_at timestamp(0) with time zone
);

CREATE INDEX IF NOT EXISTS tokens_user_id_scope_idx
  ON tokens (user_id, scope);

CREATE INDEX IF NOT EXISTS tokens_scope_expiry_revoked_idx
  ON tokens (scope, expiry, revoked_at);

CREATE INDEX IF NOT EXISTS tokens_user_id_session_id_idx
  ON tokens (user_id, session_id);

-- +goose Down
DROP TABLE IF EXISTS tokens;
