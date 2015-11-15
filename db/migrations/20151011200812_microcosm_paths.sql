
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE microcosms ADD COLUMN path ltree;

CREATE INDEX microcosms_path_gist_idx ON microcosms USING GIST(path);
CREATE INDEX microcosms_path_idx ON microcosms USING btree(path);

WITH RECURSIVE parent_microcosms AS (
    SELECT microcosm_id
          ,CAST(microcosm_id AS VARCHAR) AS path
      FROM microcosms
     WHERE parent_id IS NULL
     UNION ALL
    SELECT c.microcosm_id
          ,p.path || '.' || CAST(c.microcosm_id AS VARCHAR)
      FROM microcosms c
      JOIN parent_microcosms p ON c.parent_id = p.microcosm_id
)
UPDATE microcosms m
   SET path = CAST(p.path AS ltree)
  FROM parent_microcosms p
 WHERE m.microcosm_id = p.microcosm_id;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP INDEX microcosms_path_gist_idx;
DROP INDEX microcosms_path_idx;

ALTER TABLE microcosms DROP COLUMN path;
