
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE INDEX updates_noparent_idx ON updates (item_type_id, parent_item_type_id, parent_item_id) WHERE (item_type_id = 4 AND parent_item_type_id = 0 AND parent_item_id = 0);
CREATE INDEX updates_updatetypeid2_idx ON updates (update_type_id);
CREATE INDEX flags_itemtypeidanditemid_idx ON flags USING btree (item_type_id, item_id);

ALTER TABLE ONLY development.updates_latest DROP CONSTRAINT updates_latest_update_id_fkey;
ALTER TABLE ONLY development.updates_latest DROP CONSTRAINT updates_latest_pkey;
DROP TABLE development.updates_latest;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

CREATE TABLE updates_latest (
    update_id bigint NOT NULL
);
ALTER TABLE development.updates_latest OWNER TO microcosm;
ALTER TABLE ONLY updates_latest ADD CONSTRAINT updates_latest_pkey PRIMARY KEY (update_id);
ALTER TABLE ONLY updates_latest ADD CONSTRAINT updates_latest_update_id_fkey FOREIGN KEY (update_id) REFERENCES updates(update_id);

DROP INDEX development.flags_itemtypeidanditemid_idx;
DROP INDEX development.updates_updatetypeid2_idx;
DROP INDEX development.updates_noparent_idx;
