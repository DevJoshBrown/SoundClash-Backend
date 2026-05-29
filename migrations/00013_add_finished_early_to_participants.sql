-- +goose Up
ALTER TABLE battle_participants ADD COLUMN finished_early BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE battle_participants DROP COLUMN finished_early;
