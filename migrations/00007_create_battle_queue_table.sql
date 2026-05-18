-- +goose Up
CREATE TABLE battle_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    genres TEXT[] NOT NULL CHECK (array_length(genres, 1) >= 1),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX battle_queue_joined_at_idx ON battle_queue(joined_at);

-- +goose Down
DROP INDEX IF EXISTS battle_queue_joined_at_idx;
DROP TABLE battle_queue;
