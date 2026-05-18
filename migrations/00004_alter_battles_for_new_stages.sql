-- +goose Up
ALTER TABLE battles DROP CONSTRAINT battles_status_check;

ALTER TABLE battles ADD CONSTRAINT battles_status_check
    CHECK (status IN ('waiting','forming','in_progress','upload','listening','voting','results','cancelled'));

ALTER TABLE battles ALTER COLUMN creator_id DROP NOT NULL;

-- +goose Down
ALTER TABLE battles DROP CONSTRAINT battles_status_check;

ALTER TABLE battles ADD CONSTRAINT battles_status_check
    CHECK (status IN ('waiting','in_progress','listening','voting','completed','cancelled'));

ALTER TABLE battles ALTER COLUMN creator_id SET NOT NULL;
