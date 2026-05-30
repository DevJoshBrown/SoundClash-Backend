-- name: CreateUser :one
INSERT INTO users (username, display_name)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateUserElo :one
UPDATE users
SET elo_rating = $2
WHERE id = $1
RETURNING *;

-- name: IncrementBattlesPlayed :one
UPDATE users
SET battles_played = battles_played + 1
WHERE id = $1
RETURNING *;

-- name: IncrementBattlesWon :one
UPDATE users
SET battles_won = battles_won + 1
WHERE id = $1
RETURNING *;

-- name: GetUserByClerkID :one
SELECT * FROM users
WHERE clerk_id = $1;

-- name: UpdateUserProfile :one
UPDATE users
SET username = $2, display_name = $3
WHERE id = $1
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;

-- name: UpsertUserByClerkID :one
INSERT INTO users (username, display_name, clerk_id)
VALUES ($1, $2, $3)
ON CONFLICT (clerk_id) DO UPDATE
SET clerk_id = EXCLUDED.clerk_id,
display_name = EXCLUDED.display_name
RETURNING *;
