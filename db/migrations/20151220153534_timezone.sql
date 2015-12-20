
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE events ADD COLUMN tz_name character varying(250) NOT NULL DEFAULT 'Europe/Belfast';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE events DROP COLUMN tz_name;

