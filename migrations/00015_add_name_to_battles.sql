-- +goose Up
ALTER TABLE battles ADD COLUMN name TEXT;

-- +goose Down
ALTER TABLE battles DROP COLUMN name;
