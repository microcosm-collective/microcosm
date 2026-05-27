-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION is_banned(
    in_site_id bigint DEFAULT 0,
    in_microcosm_id bigint DEFAULT 0,
    in_profile_id bigint DEFAULT 0)
  RETURNS boolean
  LANGUAGE plpgsql
  AS
$BODY$
DECLARE
BEGIN

    IF (SELECT EXISTS(
        SELECT 1
         FROM bans b
          JOIN profiles p ON b.user_id = p.user_id
         WHERE b.site_id = in_site_id
           AND p.profile_id = in_profile_id
           AND (b.expires IS NULL OR b.expires > NOW())
    )) THEN
        RETURN true;
    END IF;

    IF in_microcosm_id = 0 THEN
        RETURN false;
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
$BODY$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION is_banned(
    in_site_id bigint DEFAULT 0,
    in_microcosm_id bigint DEFAULT 0,
    in_profile_id bigint DEFAULT 0)
  RETURNS boolean
  LANGUAGE plpgsql
  AS
$BODY$
DECLARE
BEGIN

    IF in_microcosm_id = 0 AND (SELECT EXISTS(
        SELECT 1
          FROM bans b
          JOIN profiles p ON b.user_id = p.user_id
         WHERE b.site_id = in_site_id
           AND p.profile_id = in_profile_id
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
$BODY$;
-- +goose StatementEnd
