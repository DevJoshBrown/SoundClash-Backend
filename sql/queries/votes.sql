-- name: CreateVote :one
INSERT INTO votes (battle_id, voter_id, voted_for_participant_id, score)
VALUES ($1, $2, $3, $4)
RETURNING *;


-- name: GetVotesForBattle :many
SELECT * FROM votes
WHERE battle_id = $1;

-- name: GetVoteByVoterAndParticipant :one
SELECT * FROM votes
WHERE battle_id = $1 AND voter_id = $2 AND voted_for_participant_id = $3;
