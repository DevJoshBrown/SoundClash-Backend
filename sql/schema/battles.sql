CREATE TABLE battles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID REFERENCES users(id),
    name TEXT,
    mode TEXT NOT NULL CHECK (mode IN ('sample_pack','ffa')),
    genre TEXT,
    sample_pack_id UUID,
    status TEXT NOT NULL DEFAULT 'waiting'
        CHECK (status IN ('waiting','forming','in_progress','upload','listening','voting','results','cancelled')),
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes IN (10, 15, 20, 30,60,120)),
    max_participants INTEGER NOT NULL CHECK (max_participants BETWEEN 2 AND 16),
    current_listening_index INTEGER NOT NULL DEFAULT 0,
    listening_order UUID[],
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (mode = 'sample_pack' AND sample_pack_id IS NOT NULL) OR
        (mode = 'ffa' AND genre IS NOT NULL)
    )
);
