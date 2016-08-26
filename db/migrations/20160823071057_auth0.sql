
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE sites ADD COLUMN auth0_domain character varying(100) DEFAULT NULL;
ALTER TABLE sites ADD COLUMN auth0_client_id character varying(100) DEFAULT NULL;
ALTER TABLE sites ADD COLUMN auth0_client_secret character varying(100) DEFAULT NULL;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE sites DROP COLUMN auth0_domain;
ALTER TABLE sites DROP COLUMN auth0_client_id;
ALTER TABLE sites DROP COLUMN auth0_client_secret;
