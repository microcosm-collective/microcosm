
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_microcosm_permissions_for_profile(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0,
    inprofileid bigint DEFAULT 0)
  RETURNS effective_permissions AS
$BODY$
DECLARE
    result effective_permissions;
    c effective_permissions;
BEGIN
    -- Defaults
    SELECT false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false
      INTO result.can_create,
           result.can_read,
           result.can_update,
           result.can_delete,
           result.can_close_own,
           result.can_open_own,
           result.can_read_others,
           result.is_guest,
           result.is_banned,
           result.is_owner,
           result.is_superuser,
           result.is_site_owner;

    FOR c IN
        SELECT rr.can_create
              ,rr.can_read
              ,rr.can_update
              ,rr.can_delete
              ,rr.can_close_own
              ,rr.can_open_own
              ,rr.can_read_others
              ,false
              ,false
              ,false
              ,false
              ,false
          FROM (
            SELECT get_role_profile(insiteid, r.role_id, inprofileid) AS profile_id
                  ,r.can_create
                  ,r.can_read
                  ,r.can_update
                  ,r.can_delete
                  ,r.can_close_own
                  ,r.can_open_own
                  ,r.can_read_others
              FROM roles r
             WHERE r.role_id IN (
                       SELECT *
                         FROM get_microcosm_roles (insiteid, inmicrocosmid)
                   )
               ) AS rr
          WHERE rr.profile_id = inprofileid
    LOOP
        SELECT BOOL_OR(c.can_create OR result.can_create)
              ,BOOL_OR(c.can_read OR result.can_read)
              ,BOOL_OR(c.can_update OR result.can_update)
              ,BOOL_OR(c.can_delete OR result.can_delete)
              ,BOOL_OR(c.can_close_own OR result.can_close_own)
              ,BOOL_OR(c.can_open_own OR result.can_open_own)
              ,BOOL_OR(c.can_read_others OR result.can_read_others)
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    END LOOP;

    RETURN result;

END;
$BODY$
  LANGUAGE plpgsql STABLE
  COST 100;
-- +goose StatementEnd

TRUNCATE role_members_cache;
TRUNCATE permissions_cache;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_microcosm_permissions_for_profile(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0,
    inprofileid bigint DEFAULT 0)
  RETURNS effective_permissions AS
$BODY$
DECLARE
    result effective_permissions;
    c effective_permissions;
BEGIN
    -- Defaults
    SELECT false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false,
           false
      INTO result.can_create,
           result.can_read,
           result.can_update,
           result.can_delete,
           result.can_close_own,
           result.can_open_own,
           result.can_read_others,
           result.is_guest,
           result.is_banned,
           result.is_owner,
           result.is_superuser,
           result.is_site_owner;

    FOR c IN
        SELECT rr.can_create
              ,rr.can_read
              ,rr.can_update
              ,rr.can_delete
              ,rr.can_close_own
              ,rr.can_open_own
              ,rr.can_read_others
              ,false
              ,false
              ,false
              ,false
              ,false
          FROM (
            SELECT get_role_profile(insiteid, r.role_id, inprofileid) AS profile_id
                  ,r.can_create
                  ,r.can_read
                  ,r.can_update
                  ,r.can_delete
                  ,r.can_close_own
                  ,r.can_open_own
                  ,r.can_read_others
              FROM roles r
             WHERE r.role_id IN (
                       SELECT *
                         FROM get_microcosm_roles (insiteid, inmicrocosmid)
                   )
               AND r.is_banned_role = false
               AND r.is_moderator_role = false
               ) AS rr
          WHERE rr.profile_id = inprofileid
    LOOP
        SELECT BOOL_OR(c.can_create OR result.can_create)
              ,BOOL_OR(c.can_read OR result.can_read)
              ,BOOL_OR(c.can_update OR result.can_update)
              ,BOOL_OR(c.can_delete OR result.can_delete)
              ,BOOL_OR(c.can_close_own OR result.can_close_own)
              ,BOOL_OR(c.can_open_own OR result.can_open_own)
              ,BOOL_OR(c.can_read_others OR result.can_read_others)
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    END LOOP;

    RETURN result;

END;
$BODY$
  LANGUAGE plpgsql STABLE
  COST 100;
-- +goose StatementEnd

TRUNCATE role_members_cache;
TRUNCATE permissions_cache;
