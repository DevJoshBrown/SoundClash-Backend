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

-- name: UpdateBattleStatus :one
UPDATE battles
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdateListingIndex :one
UPDATE battles
SET current_listening_index = $2
WHERE id = $1
RETURNING *;

-- name: UpdateListingOrder :one
UPDATE battles
SET listening_order = $2
WHERE id = $1
RETURNING *;
