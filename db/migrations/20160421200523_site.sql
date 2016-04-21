
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

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
    profile_id BIGINT NOT NULL,
    site_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    criteria_id BIGINT NOT NULL,
    microcosm_id BIGINT NOT NULL,
    now TIMESTAMP WITHOUT TIME ZONE NOT NULL
) ON COMMIT DROP;

INSERT INTO new_ids (
    profile_id, site_id, role_id, criteria_id, microcosm_id, now
)
SELECT NEXTVAL('profiles_profile_id_seq'),
       NEXTVAL('sites_site_id_seq'),
       NEXTVAL('roles_role_id_seq'),
       NEXTVAL('criteria_criteria_id_seq'),
       NEXTVAL('microcosms_microcosm_id_seq'),
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

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

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
