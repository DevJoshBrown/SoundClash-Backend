-- name: CreateParticipant :one
INSERT INTO battle_participants (battle_id, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetParticipant :one
SELECT * FROM battle_participants
WHERE battle_id = $1 AND user_id = $2;

-- name: ListParticipants :many
SELECT * FROM battle_participants
WHERE battle_id = $1 ORDER BY joined_at;

-- name: UpdateParticipantBeatURL :one
UPDATE battle_participants
SET beat_url = $2, submitted_at = $3
WHERE id = $1
RETURNING *;

-- name: ConfirmVotes :one
UPDATE battle_participants
SET votes_confirmed = TRUE
WHERE id = $1
RETURNING *;

-- name: GetParticipantByID :one
SELECT * FROM battle_participants
WHERE id = $1;

-- name: UpdateParticipantDuration :one
UPDATE battle_participants
SET duration_seconds = $2
WHERE id = $1
RETURNING *;

-- name: RemoveParticipant :exec
DELETE FROM battle_participants
WHERE battle_id = $1 AND user_id = $2;

-- name: MarkFinishedEarly :exec
UPDATE battle_participants
SET finished_early = TRUE
WHERE battle_id = $1 AND user_id = $2;

-- name: AllFinishedEarly :one
SELECT COUNT(*) = COUNT(*) FILTER (WHERE finished_early = TRUE)
FROM battle_participants
WHERE battle_id = $1;

-- name: UnconfirmVotes :one
UPDATE battle_participants
SET votes_confirmed = false
WHERE battle_id = $1 AND user_id = $2
RETURNING *;
