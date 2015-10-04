
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;

CREATE TABLE development.platform
(
   env character varying(20), 
   PRIMARY KEY (env)
) 
WITH (
  OIDS = FALSE
);

INSERT INTO development.platform (env) VALUES ('development');

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;

DROP TABLE development.platform;
