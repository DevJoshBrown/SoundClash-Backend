-- name: CreateUser :one
INSERT INTO users (username, display_name)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;
