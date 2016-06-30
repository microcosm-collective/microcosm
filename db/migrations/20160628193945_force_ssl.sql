
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE sites ADD COLUMN force_ssl boolean NOT NULL DEFAULT false;

COMMENT ON TABLE development.sites IS 'Basic knowledge of which sites exist. These are essentially collections of microcosms for a given URL with an assigned administrator.';


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

COMMENT ON TABLE development.sites IS 'Basic knowledge of which sites exist. These are essentially collections of microcosms for a given URL with an assigned administrator. Dull stuff.';

ALTER TABLE sites DROP COLUMN force_ssl;
