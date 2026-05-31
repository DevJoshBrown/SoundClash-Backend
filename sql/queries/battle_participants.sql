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
    bp.participant_status,
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
SET participant_status = 'finished'
WHERE battle_id = $1 AND user_id = $2;

-- name: AllFinishedEarly :one
SELECT NOT EXISTS (
    SELECT 1 FROM battle_participants
    WHERE battle_id = $1
    AND participant_status = 'active');

-- name: SetParticipantAbsent :exec
UPDATE battle_participants
SET participant_status = 'absent'
WHERE battle_id = $1 AND user_id = $2;

-- name: SetParticipantActive :exec
UPDATE battle_participants
SET participant_status = 'active'
WHERE battle_id = $1 AND user_id = $2;

-- name: SetParticipantDisqualified :exec
UPDATE battle_participants
SET participant_status = 'disqualified'
WHERE battle_id = $1 AND user_id = $2;

-- name: UnconfirmVotes :one
UPDATE battle_participants
SET votes_confirmed = false
WHERE battle_id = $1 AND user_id = $2
RETURNING *;

-- name: CountActiveParticipants :one
SELECT COUNT(*) FROM battle_participants
WHERE battle_id = $1
AND participant_status NOT IN ('disqualified', 'absent');

-- name: GetActiveParticipantForUser :one
SELECT bp.battle_id, bp.participant_status
FROM battle_participants bp
JOIN battles b ON bp.battle_id = b.id
WHERE bp.user_id = $1
AND b.status = 'in_progress'
AND bp.participant_status IN ('active','absent','finished')
LIMIT 1;
