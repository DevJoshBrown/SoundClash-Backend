-- name: CreateParticipant :one
INSERT INTO battle_participants (battle_id, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetParticipant :one
SELECT * FROM battle_participants
WHERE battle_id = $1 AND user_id = $2;

-- name: ListParticipants :many
SELECT * FROM battle_participants
WHERE battle_id = $1;

-- name: UpdateParticipantBeatURL :one
UPDATE battle_participants
SET beat_url = $2, submitted_at = $3
WHERE id = $1
RETURNING *;
