-- name: CreateBattle :one
INSERT INTO battles (creator_id, mode, genre, name, sample_pack_id, duration_minutes, max_participants)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetBattle :one
SELECT * FROM battles
WHERE id = $1;

-- name: ListBattles :many
SELECT * FROM battles
WHERE status = 'waiting'
ORDER BY created_at DESC;

-- name: UpdateBattleStatus :one
UPDATE battles
SET status = $2
WHERE id = $1
RETURNING *;

-- name: StartBattle :one
UPDATE battles
SET status = 'in_progress', started_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateListeningIndex :one
UPDATE battles
SET current_listening_index = $2
WHERE id = $1
RETURNING *;

-- name: UpdateListeningOrder :one
UPDATE battles
SET listening_order = $2
WHERE id = $1
RETURNING *;

-- name: DeleteBattleParticipants :exec
DELETE FROM battle_participants WHERE battle_id = $1;

-- name: DeleteBattle :exec
DELETE FROM battles WHERE id = $1;
