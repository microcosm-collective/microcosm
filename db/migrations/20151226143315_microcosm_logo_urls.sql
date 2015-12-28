
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE microcosms ADD COLUMN logo_url character varying(2000) DEFAULT NULL;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE microcosms DROP COLUMN logo_url;
