
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE OR REPLACE VIEW ignores_expanded AS
SELECT *
  FROM ignores
 WHERE item_type_id <> 2
 UNION
SELECT im.profile_id
      ,2
      ,o.microcosm_id
 FROM (
          SELECT i.*
                ,m.path
            FROM ignores i
                 JOIN microcosms m ON i.item_type_id = 2 AND m.microcosm_id = i.item_id
           WHERE i.item_type_id = 2
      ) AS im
      JOIN microcosms o ON o.path <@ im.path;

COMMENT ON VIEW ignores_expanded
  IS 'Expands ignored microcosms to also ignore child microcosms of an ignored microcosm';


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP VIEW ignores_expanded;
