-- +goose Up
ALTER TABLE battle_participants
ADD COLUMN votes_confirmed BOOLEAN NOT NULL DEFAULT FALSE;


-- +goose Down
ALTER TABLE battle_participants DROP COLUMN votes_confirmed;
