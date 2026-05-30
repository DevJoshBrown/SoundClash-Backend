-- name: CreateParticipant :one
INSERT INTO battle_participants (battle_id, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetParticipant :one
SELECT * FROM battle_participants
WHERE battle_id = $1 AND user_id = $2;

-- name: ListParticipants :many
SELECT
    bp.id,
    bp.battle_id,
    bp.user_id,
    bp.beat_url,
    bp.submitted_at,
    bp.votes_confirmed,
    bp.duration_seconds,
    bp.finished_early,
    bp.created_at AS created_at,
    u.display_name,
    u.username,
    u.elo_rating,
    u.battles_played,
    u.battles_won,
    u.profile_picture_url
FROM battle_participants bp
JOIN users u ON bp.user_id = u.id
WHERE bp.battle_id = $1
ORDER BY bp.created_at ASC;


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
