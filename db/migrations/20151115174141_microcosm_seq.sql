
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE development.microcosms
  ADD COLUMN sequence bigint NOT NULL DEFAULT 9999;

UPDATE microcosms m
   SET sequence = ms.sequence
  FROM (
           SELECT microcosm_id
                 ,row_number() OVER(
                      partition BY site_id
                      ORDER BY count DESC, microcosm_id
                  ) AS sequence
             FROM (
                      SELECT m.site_id
                            ,m.microcosm_id
                            ,COALESCE(
                                 (SELECT SUM(comment_count) + SUM(item_count)
                                    FROM microcosms
                                   WHERE path <@ m.path
                                     AND is_deleted IS NOT TRUE
                                     AND is_moderated IS NOT TRUE
                                 ),
                                 0
                             ) AS count
                        FROM microcosms m
                       GROUP BY m.site_id, m.microcosm_id
                       ORDER BY site_id, count DESC
                  ) AS mm
       ) ms
  WHERE m.microcosm_id = ms.microcosm_id;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE development.microcosms
 DROP COLUMN sequence;
