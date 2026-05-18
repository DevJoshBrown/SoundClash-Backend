-- name: CreateBattle :one
INSERT INTO battles (creator_id, mode, genre, sample_pack_id, duration_minutes, max_participants)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetBattle :one
SELECT * FROM battles
WHERE id = $1;

-- name: ListBattles :many
SELECT * FROM battles
ORDER BY created_at DESC;
