SET search_path = development, public, pg_catalog;

ALTER TABLE ONLY development.schema_migrations DROP CONSTRAINT schema_migrations_pkey;
DROP TABLE development.schema_migrations;

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
