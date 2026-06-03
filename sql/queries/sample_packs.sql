-- name: CreateSamplePack :one
INSERT INTO sample_packs (name, genres, file_url)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSamplePacks :many
SELECT * FROM sample_packs ORDER BY created_at DESC;

-- name: GetRandomPackByGenre :one
SELECT * FROM sample_packs
WHERE $1 = ANY(genres)
ORDER BY RANDOM()
LIMIT 1;
