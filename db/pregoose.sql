SET search_path = development, public, pg_catalog;

ALTER TABLE ONLY development.schema_migrations DROP CONSTRAINT schema_migrations_pkey;
DROP TABLE development.schema_migrations;

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

INSERT INTO development.platform (env) VALUES ('production');

SET search_path = public, pg_catalog;

CREATE SEQUENCE goose_db_version_id_seq
	START WITH 1
	INCREMENT BY 1
	NO MAXVALUE
	NO MINVALUE
	CACHE 1;

CREATE TABLE goose_db_version (
	id integer DEFAULT nextval('goose_db_version_id_seq'::regclass) NOT NULL,
	version_id bigint NOT NULL,
	is_applied boolean NOT NULL,
	tstamp timestamp without time zone DEFAULT now()
);

ALTER SEQUENCE goose_db_version_id_seq
	OWNED BY goose_db_version.id;

ALTER TABLE goose_db_version
	ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);

INSERT INTO goose_db_version (id, version_id, is_applied, tstamp) VALUES (1, 0, true, '2015-10-01 05:45:37.813543');
INSERT INTO goose_db_version (id, version_id, is_applied, tstamp) VALUES (2, 20150930224827, true, '2015-10-01 05:45:37.853826');
INSERT INTO goose_db_version (id, version_id, is_applied, tstamp) VALUES (3, 20151001060912, true, '2015-10-01 05:45:40.443675');
INSERT INTO goose_db_version (id, version_id, is_applied, tstamp) VALUES (4, 20151001065335, true, '2015-10-01 05:55:31.933328');

SELECT pg_catalog.setval('goose_db_version_id_seq', 4, true);
