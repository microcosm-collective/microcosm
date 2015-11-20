
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

DROP FUNCTION is_banned(bigint, bigint, bigint);

-- +goose StatementBegin
CREATE FUNCTION is_banned(
    in_site_id bigint DEFAULT 0,
    in_microcosm_id bigint DEFAULT 0,
    in_profile_id bigint DEFAULT 0)
  RETURNS boolean AS
$BODY$
DECLARE
BEGIN

    IF in_microcosm_id = 0 AND (SELECT EXISTS(
        SELECT 1
          FROM bans b
          JOIN profiles p ON b.user_id = p.user_id  
         WHERE b.site_id = $1 -- site_id
           AND p.profile_id = $3 -- profile_id
    )) THEN
        RETURN true;
    END IF;

    RETURN (
        WITH sr AS (
            SELECT role_id
              FROM roles
              JOIN (
                       SELECT microcosm_id
                         FROM (
                                  SELECT path
                                    FROM microcosms
                                   WHERE microcosm_id = in_microcosm_id
                              ) im
                         JOIN microcosms m ON m.path @> im.path
                   ) pm ON roles.microcosm_id = pm.microcosm_id
             WHERE is_banned_role IS TRUE
        )
        SELECT CASE WHEN COUNT(*) > 0 THEN TRUE ELSE FALSE END
          FROM profiles AS p
          JOIN sites AS s ON s.site_id = p.site_id
         WHERE s.site_id = in_site_id
           AND p.profile_id = in_profile_id
           AND p.profile_id <> s.created_by
           AND p.profile_id <> s.owned_by
           AND p.profile_id IN (
                   SELECT get_role_profiles(in_site_id, r.role_id) AS profile_id
                     FROM (SELECT * FROM sr) AS r
               )
    );
    
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
-- +goose StatementEnd

ALTER FUNCTION is_banned(bigint, bigint, bigint)
  OWNER TO microcosm;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_microcosm_regular_roles(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0)
  RETURNS SETOF bigint AS
$BODY$
DECLARE
    l_parent_id bigint;
BEGIN
    -- Are there custom roles against this microcosm?
    IF EXISTS (
        SELECT 1
          FROM roles
         WHERE site_id = insiteid
           AND microcosm_id = inmicrocosmid
           AND is_banned_role IS NOT TRUE -- override
           AND is_moderator_role IS NOT TRUE
    ) THEN
        -- custom + special + default_special
        RETURN QUERY
            SELECT role_id
              FROM roles
             WHERE site_id = insiteid
               AND microcosm_id = inmicrocosmid
               AND is_banned_role IS NOT TRUE -- override
               AND is_moderator_role IS NOT TRUE;
    ELSE
        l_parent_id := (
            SELECT parent_id
              FROM microcosms
             WHERE microcosm_id = inmicrocosmid
        );
        IF l_parent_id IS NOT NULL THEN
            -- recursion rocks, walk the parent microcosms to find the first one
            -- that has custom roles
            RETURN QUERY
                SELECT *
                  FROM get_microcosm_regular_roles(insiteid, l_parent_id);
        ELSE
            RETURN QUERY
                SELECT role_id
                  FROM roles
                 WHERE site_id = insiteid
                   AND microcosm_id = inmicrocosmid
                   AND is_banned_role IS NOT TRUE
                   AND is_moderator_role IS NOT TRUE;
        END IF;
    END IF;
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100
  ROWS 1;
-- +goose StatementEnd

ALTER FUNCTION get_microcosm_regular_roles(bigint, bigint)
  OWNER TO microcosm;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_microcosm_roles(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0)
  RETURNS SETOF bigint AS
$BODY$
DECLARE
    l_root_id BIGINT;
    l_parent_id bigint;
BEGIN
    RETURN QUERY
        WITH sr AS (
            SELECT role_id
              FROM roles
              JOIN (
                       SELECT microcosm_id
                         FROM (
                                  SELECT path
                                    FROM microcosms
                                   WHERE microcosm_id = inmicrocosmid
                              ) im
                         JOIN microcosms m ON m.path @> im.path
                   ) pm ON roles.microcosm_id = pm.microcosm_id
             WHERE is_moderator_role IS TRUE
                OR is_banned_role IS TRUE
        )
        SELECT role_id
          FROM sr
         UNION
        SELECT *
          FROM get_microcosm_regular_roles(insiteid,inmicrocosmid);
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100
  ROWS 1;
-- +goose StatementEnd

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION development.get_microcosm_roles(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0)
  RETURNS SETOF bigint AS
$BODY$
BEGIN

    IF EXISTS (SELECT 1
              FROM roles
             WHERE site_id = insiteid
               AND microcosm_id = inmicrocosmid
               AND is_banned_role IS NOT TRUE -- override
               AND is_moderator_role IS NOT TRUE
           ) THEN
        -- overrides + special + default_special
        RETURN QUERY
            SELECT role_id
              FROM roles
             WHERE site_id = insiteid
               AND (
                        -- overrides and special
                        microcosm_id = inmicrocosmid
                     OR ( -- default_special
                            microcosm_id IS NULL
                        AND (
                        	is_banned_role IS TRUE
                         OR is_moderator_role IS TRUE)
                        )
                   );
    ELSE
      IF EXISTS (SELECT 1
          FROM roles
         WHERE site_id = insiteid
           AND microcosm_id = inmicrocosmid
           AND (   is_banned_role IS TRUE -- special
          OR is_moderator_role IS TRUE)
       ) THEN

    -- special + defaults + default_special
    RETURN QUERY
        SELECT role_id
          FROM roles
         WHERE site_id = insiteid
           AND (
                   -- defaults
                   microcosm_id IS NULL
                OR ( -- special
                       microcosm_id = inmicrocosmid
                   AND (
                       is_banned_role IS TRUE
                    OR is_moderator_role IS TRUE)
                   )
               );
      ELSE
    -- defaults + default_special
    RETURN QUERY
        SELECT role_id
          FROM roles
         WHERE site_id = insiteid
           AND microcosm_id IS NULL;
      END IF;
    END IF;
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100
  ROWS 1;
-- +goose StatementEnd

DROP FUNCTION get_microcosm_regular_roles(bigint, bigint);

DROP FUNCTION is_banned(bigint, bigint, bigint);

-- +goose StatementBegin
CREATE FUNCTION is_banned(site_id bigint DEFAULT 0, microcosm_id bigint DEFAULT 0, profile_id bigint DEFAULT 0) RETURNS boolean
    LANGUAGE plpgsql
    AS $_$
DECLARE
BEGIN

    IF microcosm_id = 0 AND (SELECT EXISTS(
        SELECT 1
          FROM bans b
          JOIN profiles p ON b.user_id = p.user_id  
         WHERE b.site_id = $1 -- site_id
           AND p.profile_id = $3 -- profile_id
    )) THEN
        RETURN true;
    END IF;


    IF (SELECT COUNT(*) FROM roles rr WHERE rr.microcosm_id = $2) > 0 THEN
        RETURN (SELECT CASE WHEN COUNT(*) > 0 THEN true ELSE false END
          FROM profiles AS pp
          JOIN sites AS ss ON ss.site_id = pp.site_id
         WHERE ss.site_id = $1 -- site_id
           AND pp.profile_id = $3 -- profile_id
           AND pp.profile_id <> ss.created_by
           AND pp.profile_id <> ss.owned_by
           AND (
                   pp.profile_id IN (
                       SELECT get_role_profiles($1, r.role_id) AS profile_id
                         FROM roles AS r
                        WHERE r.site_id = $1 -- site_id
                          AND r.is_banned_role = true
                          AND r.microcosm_id = $2 -- microcosm_id
                   )
                OR pp.profile_id IN (
                       SELECT p.profile_id
                         FROM users u
                             ,bans b
                             ,profiles p
                        WHERE u.user_id = b.user_id
                          AND b.site_id = $1 -- site_id
                          AND p.user_id = u.user_id
                   )
               )
        );
    ELSE
        RETURN (
        SELECT CASE WHEN COUNT(*) > 0 THEN true ELSE false END
          FROM profiles AS pp
          JOIN sites AS ss ON ss.site_id = pp.site_id
         WHERE ss.site_id = $1 -- site_id
           AND pp.profile_id = $3 -- profile_id
           AND pp.profile_id <> ss.created_by
           AND pp.profile_id <> ss.owned_by
           AND (
                   pp.profile_id IN (
                       SELECT get_role_profiles($1, r.role_id) AS profile_id
                         FROM roles AS r
                        WHERE r.site_id = $1 -- site_id
                          AND r.is_banned_role = true
                          AND r.microcosm_id IS NULL -- microcosm_id
                   )
                OR pp.profile_id IN (
                       SELECT p.profile_id
                         FROM users u
                             ,bans b
                             ,profiles p
                        WHERE u.user_id = b.user_id
                          AND b.site_id = $1 -- site_id
                          AND p.user_id = u.user_id
                   )
               )
        );
    END IF;
    
END;
$_$;
-- +goose StatementEnd
