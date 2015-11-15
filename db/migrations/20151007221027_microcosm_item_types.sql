
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE microcosms
  ADD COLUMN item_types bigint[] NOT NULL DEFAULT '{2,6,9}';
COMMENT ON COLUMN microcosms.item_types IS 'Array of item_types.item_type_id that are allowed to be posted in this microcosm.

Allowing just item_type_id = 2 would mean that a Microcosm is just a category as it could contain no content.

Allowing 6,9 would mean that it is not able to have child forums and is a leaf forum.

Allowing 9 would mean that it can only contain events.

2,6,9 would currently allow everything.';


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE microcosms
  DROP COLUMN item_types;
