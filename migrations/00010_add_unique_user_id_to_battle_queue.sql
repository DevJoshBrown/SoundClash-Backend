-- +goose Up
ALTER TABLE battle_queue ADD CONSTRAINT battle_queue_user_id_unique UNIQUE (user_id);

-- +goose Down
ALTER TABLE battle_queue DROP CONSTRAINT battle_queue_user_id_unique;
