-- +goose Up
ALTER TABLE tokens
  ADD CONSTRAINT tokens_refresh_session_id_check CHECK (
    (scope = 'refresh' AND session_id IS NOT NULL)
    OR
    (scope <> 'refresh' AND session_id IS NULL)
  );

-- +goose Down
ALTER TABLE tokens
  DROP CONSTRAINT IF EXISTS tokens_refresh_session_id_check;
