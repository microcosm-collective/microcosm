
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION has_unread(
    in_item_type_id bigint DEFAULT 0,
    in_item_id bigint DEFAULT 0,
    in_profile_id bigint DEFAULT 0)
  RETURNS boolean AS
$BODY$
DECLARE
BEGIN

    IF in_profile_id = 0 THEN
        RETURN false;
    END IF;

    CASE in_item_type_id
    WHEN 1 THEN -- site
    WHEN 2 THEN -- microcosm

        -- Check child forums should they exist
        IF (SELECT COALESCE(BOOL_OR(has_unread(2, m.microcosm_id, in_profile_id)), FALSE)
              FROM microcosms m
              LEFT JOIN permissions_cache p ON p.site_id = m.site_id
                                           AND p.item_type_id = 2
                                           AND p.item_id = m.microcosm_id
                                           AND p.profile_id = in_profile_id
                   LEFT JOIN ignores_expanded i ON i.profile_id = in_profile_id
                                               AND i.item_type_id = 2
                                               AND i.item_id = m.microcosm_id
             WHERE m.parent_id = in_item_id
               AND m.is_deleted IS NOT TRUE
               AND m.is_moderated IS NOT TRUE
               AND i.profile_id IS NULL
               AND (
                       (p.can_read IS NOT NULL AND p.can_read IS TRUE)
                    OR (get_effective_permissions(m.site_id,m.microcosm_id,2,m.microcosm_id,in_profile_id)).can_read IS TRUE
                   )) THEN
            RETURN TRUE;
        END IF;

        -- Check the last read of the microcosm against the read time
        IF NOT (SELECT COALESCE(
                (
                    SELECT last_modified
                      FROM flags
                     WHERE microcosm_id = in_item_id
                       AND item_type_id IN (6, 9)
                       AND NOT item_is_deleted
                       AND NOT item_is_moderated
                       AND last_modified > (
                               SELECT read
                                 FROM read
                                WHERE profile_id = in_profile_id
                                  AND item_type_id = in_item_type_id
                                  AND item_id = in_item_id
                           )
                     ORDER BY last_modified DESC
                     LIMIT 1
                ) > (
                    SELECT read
                      FROM read
                     WHERE profile_id = in_profile_id
                       AND item_type_id = in_item_type_id
                       AND item_id = in_item_id
                ), true)) THEN
            RETURN false;
        END IF;

        -- We don't have a recent last_read indicator, and need to call
        -- has_unread for items... but if we do have an old read row for
        -- the microcosm then we only need to check the items since that
        -- time.
        IF (SELECT COUNT(*)
              FROM read
             WHERE profile_id = in_profile_id
               AND item_type_id = in_item_type_id
               AND item_id = in_item_id ) > 0 THEN

            RETURN (SELECT EXISTS(
                        SELECT 1
                          FROM flags
                         WHERE microcosm_id = in_item_id
                           AND item_type_id IN (6, 9)
                           AND NOT item_is_deleted
                           AND NOT item_is_moderated
                           AND last_modified > (
                                   SELECT read
                                     FROM read
                                    WHERE profile_id = in_profile_id
                                      AND item_type_id = in_item_type_id
                                      AND item_id = in_item_id
                               )
                           AND has_unread(item_type_id, item_id, in_profile_id)
                         ORDER BY last_modified DESC
                   ));

        ELSE

            -- The really slow way, iterate every item until we hit something unread
            RETURN (SELECT EXISTS(
                        SELECT 1
                          FROM flags
                         WHERE microcosm_id = in_item_id
                           AND item_type_id IN (6, 9)
                           AND NOT item_is_deleted
                           AND NOT item_is_moderated
                           AND has_unread(item_type_id, item_id, in_profile_id)
                         ORDER BY last_modified DESC
                   ));
        END IF;

    WHEN 3 THEN -- profile
    WHEN 4 THEN -- comment
    WHEN 5 THEN -- huddle

        -- Check the last read of all huddles against the read time
        IF NOT (SELECT COALESCE(
                (
                    SELECT last_modified
                      FROM flags
                     WHERE item_type_id = 5
                       AND item_id = in_item_id
                       AND NOT item_is_deleted
                       AND NOT item_is_moderated
                       AND last_modified > (
                               SELECT read
                                 FROM read
                                WHERE profile_id = in_profile_id
                                  AND item_type_id = 5
                                  AND item_id = 0
                           )
                ) > (
                    SELECT read
                      FROM read
                     WHERE profile_id = in_profile_id
                       AND item_type_id = 5
                       AND item_id = 0
                ), true)) THEN
            RETURN false;
        END IF;


        -- We don't have a recent last_read indicator, and need to call
        -- has_unread for items... but if we do have an old read row for
        -- all huddles then we only need to check the items since that
        -- time.
        IF (SELECT EXISTS(
            SELECT 1
              FROM read
             WHERE profile_id = in_profile_id
               AND item_type_id = 5
               AND item_id = 0 )) THEN

            RETURN (SELECT EXISTS(
                       SELECT 1
                         FROM (            
                        SELECT COALESCE(f.last_modified > GREATEST(MAX(r.read), r2.read), true) AS unread
                          FROM flags f
                               LEFT JOIN read r ON r.item_type_id = 5
                                               AND r.item_id = in_item_id
                                               AND r.profile_id = in_profile_id
                              ,(
                                   SELECT read
                                     FROM read
                                    WHERE profile_id = in_profile_Id
                                      AND item_type_id = 5
                                      AND item_id = 0
                               ) r2
                         WHERE f.item_type_id = 5
                           AND f.item_id = in_item_id
                           AND f.item_is_deleted IS NOT TRUE
                           AND f.item_is_moderated IS NOT TRUE
                         GROUP BY f.last_modified, r2.read
                               ) as u
                         WHERE unread
                   ));

        ELSE

            -- The really slow way, iterate every item until we hit something unread
            RETURN (SELECT EXISTS(
                        SELECT 1
                          FROM (
                                SELECT COALESCE(i.last_modified > MAX(r.read), true) AS unread
                                  FROM flags i
                                       LEFT JOIN read r ON r.item_type_id = in_item_type_id
                                                       AND r.item_id = in_item_id
                                                       AND r.profile_id = in_profile_id
                                 WHERE i.item_type_id = in_item_type_id
                                   AND i.item_id = in_item_id
                                 GROUP BY r.read, i.last_modified
                               ) AS u
                         WHERE unread
                   ));

        END IF;

    WHEN 6 THEN -- conversation

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 7 THEN -- poll

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 8 THEN -- article

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 9 THEN -- event

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 10 THEN -- question

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 11 THEN -- classified

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 12 THEN -- album

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 13 THEN -- attendee
    WHEN 14 THEN -- user
    WHEN 15 THEN -- attribute
    WHEN 16 THEN -- update
    WHEN 17 THEN -- role
    WHEN 18 THEN -- update type
    WHEN 19 THEN -- watcher
    END CASE;

    RETURN false;

END;
$BODY$
  LANGUAGE plpgsql STABLE
  COST 100;
-- +goose StatementEnd

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION has_unread(
    in_item_type_id bigint DEFAULT 0,
    in_item_id bigint DEFAULT 0,
    in_profile_id bigint DEFAULT 0)
  RETURNS boolean AS
$BODY$
DECLARE
BEGIN

    IF in_profile_id = 0 THEN
        RETURN false;
    END IF;

    CASE in_item_type_id
    WHEN 1 THEN -- site
    WHEN 2 THEN -- microcosm

        -- Check the last read of the microcosm against the read time
        IF NOT (SELECT COALESCE(
                (
                    SELECT last_modified
                      FROM flags
                     WHERE microcosm_id = in_item_id
                       AND item_type_id IN (6, 9)
                       AND NOT item_is_deleted
                       AND NOT item_is_moderated
                       AND last_modified > (
                               SELECT read
                                 FROM read
                                WHERE profile_id = in_profile_id
                                  AND item_type_id = in_item_type_id
                                  AND item_id = in_item_id
                           )
                     ORDER BY last_modified DESC
                     LIMIT 1
                ) > (
                    SELECT read
                      FROM read
                     WHERE profile_id = in_profile_id
                       AND item_type_id = in_item_type_id
                       AND item_id = in_item_id
                ), true)) THEN
            RETURN false;
        END IF;

        -- We don't have a recent last_read indicator, and need to call
        -- has_unread for items... but if we do have an old read row for
        -- the microcosm then we only need to check the items since that
        -- time.
        IF (SELECT COUNT(*)
              FROM read
             WHERE profile_id = in_profile_id
               AND item_type_id = in_item_type_id
               AND item_id = in_item_id ) > 0 THEN

            RETURN (SELECT EXISTS(
                        SELECT 1
                          FROM flags
                         WHERE microcosm_id = in_item_id
                           AND item_type_id IN (6, 9)
                           AND NOT item_is_deleted
                           AND NOT item_is_moderated
                           AND last_modified > (
                                   SELECT read
                                     FROM read
                                    WHERE profile_id = in_profile_id
                                      AND item_type_id = in_item_type_id
                                      AND item_id = in_item_id
                               )
                           AND has_unread(item_type_id, item_id, in_profile_id)
                         ORDER BY last_modified DESC
                   ));

        ELSE

            -- The really slow way, iterate every item until we hit something unread
            RETURN (SELECT EXISTS(
                        SELECT 1
                          FROM flags
                         WHERE microcosm_id = in_item_id
                           AND item_type_id IN (6, 9)
                           AND NOT item_is_deleted
                           AND NOT item_is_moderated
                           AND has_unread(item_type_id, item_id, in_profile_id)
                         ORDER BY last_modified DESC
                   ));
        END IF;

    WHEN 3 THEN -- profile
    WHEN 4 THEN -- comment
    WHEN 5 THEN -- huddle

        -- Check the last read of all huddles against the read time
        IF NOT (SELECT COALESCE(
                (
                    SELECT last_modified
                      FROM flags
                     WHERE item_type_id = 5
                       AND item_id = in_item_id
                       AND NOT item_is_deleted
                       AND NOT item_is_moderated
                       AND last_modified > (
                               SELECT read
                                 FROM read
                                WHERE profile_id = in_profile_id
                                  AND item_type_id = 5
                                  AND item_id = 0
                           )
                ) > (
                    SELECT read
                      FROM read
                     WHERE profile_id = in_profile_id
                       AND item_type_id = 5
                       AND item_id = 0
                ), true)) THEN
            RETURN false;
        END IF;


        -- We don't have a recent last_read indicator, and need to call
        -- has_unread for items... but if we do have an old read row for
        -- all huddles then we only need to check the items since that
        -- time.
        IF (SELECT EXISTS(
            SELECT 1
              FROM read
             WHERE profile_id = in_profile_id
               AND item_type_id = 5
               AND item_id = 0 )) THEN

            RETURN (SELECT EXISTS(
                       SELECT 1
                         FROM (            
                        SELECT COALESCE(f.last_modified > GREATEST(MAX(r.read), r2.read), true) AS unread
                          FROM flags f
                               LEFT JOIN read r ON r.item_type_id = 5
                                               AND r.item_id = in_item_id
                                               AND r.profile_id = in_profile_id
                              ,(
                                   SELECT read
                                     FROM read
                                    WHERE profile_id = in_profile_Id
                                      AND item_type_id = 5
                                      AND item_id = 0
                               ) r2
                         WHERE f.item_type_id = 5
                           AND f.item_id = in_item_id
                           AND f.item_is_deleted IS NOT TRUE
                           AND f.item_is_moderated IS NOT TRUE
                         GROUP BY f.last_modified, r2.read
                               ) as u
                         WHERE unread
                   ));

        ELSE

            -- The really slow way, iterate every item until we hit something unread
            RETURN (SELECT EXISTS(
                        SELECT 1
                          FROM (
                                SELECT COALESCE(i.last_modified > MAX(r.read), true) AS unread
                                  FROM flags i
                                       LEFT JOIN read r ON r.item_type_id = in_item_type_id
                                                       AND r.item_id = in_item_id
                                                       AND r.profile_id = in_profile_id
                                 WHERE i.item_type_id = in_item_type_id
                                   AND i.item_id = in_item_id
                                 GROUP BY r.read, i.last_modified
                               ) AS u
                         WHERE unread
                   ));

        END IF;

    WHEN 6 THEN -- conversation

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 7 THEN -- poll

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 8 THEN -- article

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 9 THEN -- event

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 10 THEN -- question

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 11 THEN -- classified

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 12 THEN -- album

        RETURN COUNT(*) > 0 AS has_unread
          FROM (
                SELECT COALESCE(i.last_modified > r.read, true) AS unread
                  FROM flags i
                       LEFT JOIN "read" r
                         ON (
                                (r.item_type_id = i.item_type_id AND r.item_id = i.item_id) 
                             OR (r.item_type_id = 2 AND r.item_id = i.microcosm_id)
                            )
                        AND r.profile_id = in_profile_id
                 WHERE i.item_type_id = in_item_type_id
                   AND i.item_id = in_item_id
                 ORDER BY r.read DESC
                 LIMIT 1
               ) AS u
         WHERE unread;

    WHEN 13 THEN -- attendee
    WHEN 14 THEN -- user
    WHEN 15 THEN -- attribute
    WHEN 16 THEN -- update
    WHEN 17 THEN -- role
    WHEN 18 THEN -- update type
    WHEN 19 THEN -- watcher
    END CASE;

    RETURN false;

END;
$BODY$
  LANGUAGE plpgsql STABLE
  COST 100;
-- +goose StatementEnd
