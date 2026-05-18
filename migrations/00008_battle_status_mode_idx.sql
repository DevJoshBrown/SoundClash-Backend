-- +goose Up
CREATE INDEX battles_status_mode_idx ON battles(status, mode);

-- +goose Down
DROP INDEX IF EXISTS battles_status_mode_idx;
