-- +goose Up
CREATE TABLE battles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES users(id),
    mode TEXT NOT NULL CHECK (mode IN ('sample_pack','ffa')),
    genre TEXT,
    sample_pack_id UUID,
    status TEXT NOT NULL DEFAULT 'waiting'
        CHECK (status IN ('waiting','in_progress','listening','voting','completed','cancelled')),
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes BETWEEN 10 AND 60),
    max_participants INTEGER NOT NULL CHECK (max_participants BETWEEN 2 AND 16),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (mode = 'sample_pack' AND sample_pack_id IS NOT NULL) OR
        (mode = 'ffa' AND genre IS NOT NULL)
    )
);


-- +goose Down
DROP TABLE battles;
