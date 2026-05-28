-- +goose Up
ALTER TABLE battle_participants
ADD COLUMN duration_seconds INTEGER;

-- +goose Down
ALTER TABLE battle_participants
DROP COLUMN duration_seconds;
