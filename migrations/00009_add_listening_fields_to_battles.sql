-- +goose Up
ALTER TABLE battles
ADD COLUMN current_listening_index INTEGER NOT NULL DEFAULT 0,
ADD COLUMN listening_order UUID[];

-- +goose Down
ALTER TABLE battles
DROP COLUMN current_listening_index,
DROP COLUMN listening_order;
