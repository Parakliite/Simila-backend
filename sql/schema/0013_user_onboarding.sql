-- +goose Up
ALTER TABLE users
ADD COLUMN onboarded BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users
DROP COLUMN IF EXISTS onboarded;
