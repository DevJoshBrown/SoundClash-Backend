-- name: EnqueueUser :one
INSERT INTO battle_queue (user_id, genres)
VALUES ($1, $2)
ON CONFLICT (user_id) DO NOTHING
RETURNING *;

-- name: DequeueUser :exec
DELETE FROM battle_queue
WHERE user_id = $1;

-- name: GetQueueTicket :one
SELECT * FROM battle_queue
WHERE user_id = $1;

-- name: ListQueueTickets :many
SELECT * FROM battle_queue
ORDER BY joined_at ASC;

-- name: DeleteQueueTicketsByIDs :exec
DELETE FROM battle_queue
WHERE id = ($1::uuid[]);
