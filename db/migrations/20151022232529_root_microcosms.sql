
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Create root microcosms
INSERT INTO microcosms (
    site_id, visibility, title, description, created,
    created_by, owned_by, item_types
)
SELECT site_id
      ,'public'
      ,'Forums'
      ,description
      ,created
      ,created_by
      ,owned_by
      ,'{2,6,9}'::bigint[]
  FROM sites;

-- Assign existing microcosms to root
UPDATE microcosms m2
   SET parent_id = r.microcosm_id
  FROM (
           SELECT s.site_id
                 ,m.microcosm_id
             FROM sites s
                 ,microcosms m
            WHERE s.site_id = m.site_id
              AND s.created = m.created
              AND m.parent_id IS NULL
       ) AS r
 WHERE m2.microcosm_id != r.microcosm_id
   AND m2.site_id = r.site_id
   AND m2.parent_id IS NULL;

-- Update all paths
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
 WHERE m.microcosm_id = p.microcosm_id
   AND (
           m.path IS NULL
        OR CAST(m.path AS VARCHAR) <> p.path
       );

-- update search indexes
UPDATE search_index si
   SET microcosm_id = m.parent_id
  FROM microcosms m
 WHERE si.item_type_id = 2
   AND si.item_id = m.microcosm_id
   AND m.parent_id IS NOT NULL;

-- update flags
UPDATE flags f
   SET microcosm_id = m.parent_id
  FROM microcosms m
 WHERE f.item_type_id = 2
   AND f.item_id = m.microcosm_id
   AND m.parent_id IS NOT NULL;

-- assign default roles to root microcosms
UPDATE roles r
   SET microcosm_id = m.microcosm_id
  FROM microcosms m
 WHERE r.site_id = m.site_id
   AND m.parent_id IS NULL;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION create_owned_site(
    IN in_title CHARACTER VARYING,
    IN in_subdomain_key CHARACTER VARYING,
    IN in_theme_id BIGINT,
    IN in_user_id BIGINT,
    IN in_profile_name CHARACTER VARYING,
    IN in_avatar_id BIGINT,
    IN in_avatar_url CHARACTER VARYING,
    IN in_domain CHARACTER VARYING DEFAULT NULL::CHARACTER VARYING,
    IN in_description TEXT DEFAULT NULL::TEXT,
    IN in_logo_url CHARACTER VARYING DEFAULT NULL::CHARACTER VARYING,
    IN in_background_url CHARACTER VARYING DEFAULT NULL::CHARACTER VARYING,
    IN in_background_position CHARACTER VARYING DEFAULT NULL::CHARACTER VARYING,
    IN in_background_color CHARACTER VARYING DEFAULT NULL::CHARACTER VARYING,
    IN in_link_color CHARACTER VARYING DEFAULT NULL::CHARACTER VARYING,
    IN in_ga_web_property_id CHARACTER VARYING DEFAULT NULL::CHARACTER VARYING)
  RETURNS TABLE(new_site_id BIGINT, new_profile_id BIGINT) AS
$BODY$
DECLARE
BEGIN

DROP TABLE IF EXISTS new_ids;

CREATE TEMP TABLE new_ids (
    profile_id BIGINT NOT NULL,
    site_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    criteria_id BIGINT NOT NULL,
    microcosm_id BIGINT NOT NULL,
    now TIMESTAMP WITHOUT TIME ZONE NOT NULL
) ON COMMIT DROP;

INSERT INTO new_ids (
    profile_id, site_id, role_id, criteria_id, microcosm_id
)
SELECT NEXTVAL('profiles_profile_id_seq'),
       NEXTVAL('sites_site_id_seq'),
       NEXTVAL('roles_role_id_seq'),
       NEXTVAL('criteria_criteria_id_seq'),
       NEXTVAL('microcosm_microcosm_id_seq'),
       NOW();

INSERT INTO sites (
    site_id, title, description, subdomain_key, domain,
    created, created_by, owned_by, theme_id, logo_url,
    background_url, background_position, background_color, link_color, ga_web_property_id
)
SELECT site_id, in_title, in_description, in_subdomain_key, in_domain,
       now, profile_id, profile_id, in_theme_id, in_logo_url,
       in_background_url, in_background_position, in_background_color, in_link_color, in_ga_web_property_id
  FROM new_ids;

INSERT INTO profiles (
    profile_id, site_id, user_id, profile_name, created,
    last_active, avatar_id, avatar_url
)
SELECT profile_id, site_id, in_user_id, in_profile_name, now,
       now, in_avatar_id, in_avatar_url
  FROM new_ids;

INSERT INTO microcosms (
    microcosm_id, site_id, visibility, title, description,
    created, created_by, owned_by, item_types
)
SELECT microcosm_id, site_id, 'public', 'Forums', in_description,
       now, profile_id, profile_id, '{2,6,9}'::BIGINT[]
  FROM new_ids;

INSERT INTO roles (
    role_id, title, site_id, microcosm_id, created,
    created_by, is_moderator_role, is_banned_role, include_guests, can_read,
    can_create, can_update, can_delete, can_close_own, can_open_own,
    can_read_others
)
SELECT role_id, 'Guests', site_id, microcosm_id, now,
       NULL, false, false, true, true,
       false, false, false, false, false,
       false
  FROM new_ids;

UPDATE new_ids SET role_id = NEXTVAL('roles_role_id_seq');
INSERT INTO roles (
    role_id, title, site_id, microcosm_id, created,
    created_by, is_moderator_role, is_banned_role, include_guests, can_read,
    can_create, can_update, can_delete, can_close_own, can_open_own,
    can_read_others, include_users
)
SELECT role_id, 'Members', site_id, microcosm_id, now,
       NULL, false, false, false, true,
       true, false, false, true, false,
       true, true
  FROM new_ids;

UPDATE new_ids SET role_id = NEXTVAL('roles_role_id_seq');
INSERT INTO roles (
    role_id, title, site_id, microcosm_id, created,
    created_by, is_moderator_role, is_banned_role, include_guests, can_read,
    can_create, can_update, can_delete, can_close_own, can_open_own,
    can_read_others
) SELECT
    role_id, 'Banned', site_id, microcosm_id, now,
    NULL, false, true, false, false,
    false, false, false, false, false,
    false
FROM new_ids;

INSERT INTO site_options (
    site_id, send_email, send_sms, only_admins_can_create_microcosms
)
SELECT site_id, true, false, true
FROM new_ids;

-- To get around the chicken and egg problem of foreign keys
-- the trigger to update flags used the lowest valid profile_id
-- which we now correct.
UPDATE roles r
   SET created_by = n.profile_id
  FROM new_ids n
 WHERE r.site_id = n.site_id;

UPDATE flags f
   SET created_by = n.profile_id
  FROM new_ids n
 WHERE f.item_type_id = 1
   AND f.item_id = n.site_id;

RETURN QUERY SELECT site_id, profile_id FROM new_ids;
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100
  ROWS 1000;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION development.get_microcosm_roles(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0)
  RETURNS SETOF bigint AS
$BODY$
DECLARE
    l_root_id BIGINT;
    l_parent_id bigint;
BEGIN
    -- Get the id of the root microcosm as this holds our default roles
    l_root_id := (
        SELECT microcosm_id
          FROM microcosms
         WHERE site_id = insiteid
           AND parent_id IS NULL
    );
    -- Are there custom roles against this microcosm?
    IF EXISTS (
        SELECT 1
          FROM roles
         WHERE site_id = insiteid
           AND microcosm_id = inmicrocosmid
           AND microcosm_id != l_root_id
           AND is_banned_role IS NOT TRUE -- override
           AND is_moderator_role IS NOT TRUE
    ) THEN
        -- custom + special + default_special
        RETURN QUERY
            SELECT role_id
              FROM roles
             WHERE site_id = insiteid
               AND (
                       -- overrides + special
                       microcosm_id = inmicrocosmid
                    OR (
                           -- default_special
                           microcosm_id = l_root_id
                       AND (
                               is_banned_role IS TRUE
                            OR is_moderator_role IS TRUE
                           )
                       )
                   );
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
                  FROM get_microcosm_roles(insiteid, l_parent_id);
        ELSE
            IF EXISTS (
                SELECT 1
                  FROM roles
                 WHERE site_id = insiteid
                   AND microcosm_id = inmicrocosmid
                   AND (
                           is_banned_role IS TRUE -- special
                        OR is_moderator_role IS TRUE
                       )
            ) THEN
                -- special + defaults + default_special
                RETURN QUERY
                    SELECT role_id
                      FROM roles
                     WHERE site_id = insiteid
                       AND (
                               -- defaults
                               microcosm_id = l_root_id
                            OR (   -- special
                                   microcosm_id = inmicrocosmid
                               AND (
                                       is_banned_role IS TRUE
                                    OR is_moderator_role IS TRUE
                                   )
                               )
                           );
            ELSE
                -- defaults + default_special
                RETURN QUERY
                    SELECT role_id
                      FROM roles
                     WHERE site_id = insiteid
                       AND microcosm_id = l_root_id;
            END IF;
        END IF;
    END IF;
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100
  ROWS 1;
-- +goose StatementEnd

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- There is no effective downgrade from this as we will not perform a downgrade
-- that could be destructive, and this potentially would be (we'd need to nuke
-- any content in the root microcosm that wasn't a microcosm)

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION create_owned_site(
    IN in_title character varying,
    IN in_subdomain_key character varying,
    IN in_theme_id bigint,
    IN in_user_id bigint,
    IN in_profile_name character varying,
    IN in_avatar_id bigint,
    IN in_avatar_url character varying,
    IN in_domain character varying DEFAULT NULL::character varying,
    IN in_description text DEFAULT NULL::text,
    IN in_logo_url character varying DEFAULT NULL::character varying,
    IN in_background_url character varying DEFAULT NULL::character varying,
    IN in_background_position character varying DEFAULT NULL::character varying,
    IN in_background_color character varying DEFAULT NULL::character varying,
    IN in_link_color character varying DEFAULT NULL::character varying,
    IN in_ga_web_property_id character varying DEFAULT NULL::character varying)
  RETURNS TABLE(new_site_id bigint, new_profile_id bigint) AS
$BODY$
DECLARE
BEGIN

DROP TABLE IF EXISTS new_ids;

CREATE TEMP TABLE new_ids (
    profile_id bigint NOT NULL,
    site_id bigint NOT NULL,
    role_id bigint NOT NULL,
    criteria_id bigint NOT NULL
) ON COMMIT DROP;

INSERT INTO new_ids (profile_id, site_id, role_id, criteria_id)
SELECT NEXTVAL('profiles_profile_id_seq'),
       NEXTVAL('sites_site_id_seq'),
       NEXTVAL('roles_role_id_seq'),
       NEXTVAL('criteria_criteria_id_seq');

INSERT INTO sites (
    site_id,
    title,
    description,
    subdomain_key,
    domain,
    
    created,
    created_by,
    owned_by,
    theme_id,
    logo_url,
    
    background_url,
    background_position,
	background_color,
	link_color,
    ga_web_property_id
) SELECT 
    site_id,
    in_title,
    in_description,
    in_subdomain_key,
    in_domain,
    
    NOW(),
    profile_id,
    profile_id,
    in_theme_id,
    in_logo_url,
    
    in_background_url,
    in_background_position,
	in_background_color,
	in_link_color,
    in_ga_web_property_id
FROM new_ids;

INSERT INTO profiles (
    profile_id,
    site_id,
    user_id,
    profile_name,
    created,

    last_active,
    avatar_id,
    avatar_url
) SELECT
    profile_id,
    site_id,
    in_user_id,
    in_profile_name,
    NOW(),

    NOW(),
    in_avatar_id,
    in_avatar_url
FROM new_ids;

INSERT INTO roles (
    role_id,
    title,
    site_id,
    microcosm_id,
    created,

    created_by,
    is_moderator_role,
    is_banned_role,
    include_guests,
    can_read,

    can_create,
    can_update,
    can_delete,
    can_close_own,
    can_open_own,

    can_read_others
) SELECT 
    role_id,
    'Guests',
    site_id,
    NULL,
    NOW(),

    NULL,
    false,
    false,
    true,
    true,

    false,
    false,
    false,
    false,
    false,

    false
FROM new_ids;

UPDATE new_ids SET role_id = NEXTVAL('roles_role_id_seq');
INSERT INTO roles (
    role_id,
    title,
    site_id,
    microcosm_id,
    created,

    created_by,
    is_moderator_role,
    is_banned_role,
    include_guests,
    can_read,

    can_create,
    can_update,
    can_delete,
    can_close_own,
    can_open_own,

    can_read_others,
    include_users
) SELECT
    role_id,
    'Members',
    site_id,
    NULL,
    NOW(),

    NULL,
    false,
    false,
    false,
    true,

    true,
    false,
    false,
    true,
    false,

    true,
    TRUE
FROM new_ids;

UPDATE new_ids SET role_id = NEXTVAL('roles_role_id_seq');
INSERT INTO roles (
    role_id,
    title,
    site_id,
    microcosm_id,
    created,

    created_by,
    is_moderator_role,
    is_banned_role,
    include_guests,
    can_read,

    can_create,
    can_update,
    can_delete,
    can_close_own,
    can_open_own,

    can_read_others
) SELECT
    role_id,
    'Banned',
    site_id,
    NULL,
    NOW(),

    NULL,
    false,
    true,
    false,
    false,

    false,
    false,
    false,
    false,
    false,

    false
FROM new_ids;

INSERT INTO site_options (
    site_id,
    send_email,
    send_sms,
    only_admins_can_create_microcosms
)
SELECT
    site_id,
    true,
    false,
    true
FROM new_ids;

-- To get around the chicken and egg problem of foreign keys
-- the trigger to update flags used the lowest valid profile_id
-- which we now correct.
UPDATE roles r
   SET created_by = n.profile_id
  FROM new_ids n
 WHERE r.site_id = n.site_id;

UPDATE flags f
   SET created_by = n.profile_id
  FROM new_ids n
 WHERE f.item_type_id = 1
   AND f.item_id = n.site_id;

RETURN QUERY SELECT site_id, profile_id FROM new_ids;
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100
  ROWS 1000;
-- +goose StatementEnd

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
         AND (   is_banned_role IS TRUE
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
           AND (   is_banned_role IS TRUE
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
