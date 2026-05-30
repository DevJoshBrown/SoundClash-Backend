CREATE TABLE users (
    id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    elo_rating INTEGER NOT NULL DEFAULT 1000,
    battles_played INTEGER NOT NULL DEFAULT 0,
    battles_won INTEGER NOT NULL DEFAULT 0,
    clerk_id TEXT UNIQUE,
    profile_picture_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

);
