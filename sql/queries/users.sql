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
