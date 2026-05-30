CREATE TABLE battle_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    battle_id UUID NOT NULL REFERENCES battles(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    beat_url TEXT,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_seconds INTEGER,
    submitted_at TIMESTAMPTZ,
    votes_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    finished_early BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (battle_id, user_id)
);
