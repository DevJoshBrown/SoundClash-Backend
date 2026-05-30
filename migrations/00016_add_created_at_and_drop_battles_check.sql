-- +goose Up
ALTER TABLE battle_participants ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE battles DROP CONSTRAINT battles_check;

-- +goose Down
ALTER TABLE battles ADD CONSTRAINT battles_check CHECK (
    (mode = 'sample_pack' AND sample_pack_id IS NOT NULL) OR
    (mode = 'ffa' AND genre IS NOT NULL)
);
