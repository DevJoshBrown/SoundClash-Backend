-- +goose Up
ALTER TABLE battle_participants
ADD COLUMN participant_status TEXT NOT NULL DEFAULT 'active'
    CHECK(participant_status IN ('active','finished','absent','disqualified')),
DROP COLUMN finished_early;

-- +goose Down
ALTER TABLE battle_participants
DROP COLUMN participant_status,
ADD COLUMN finished_early BOOLEAN NOT NULL DEFAULT FALSE;
