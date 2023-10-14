
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;

CREATE SCHEMA development;

ALTER SCHEMA development OWNER TO microcosm;

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;

SET search_path = development, public, pg_catalog;

ALTER ROLE microcosm SET search_path TO development, public;

CREATE TYPE effective_permissions AS (
	can_create boolean,
	can_read boolean,
	can_update boolean,
	can_delete boolean,
	can_close_own boolean,
	can_open_own boolean,
	can_read_others boolean,
	is_guest boolean,
	is_banned boolean,
	is_owner boolean,
	is_superuser boolean,
	is_site_owner boolean
);

ALTER TYPE development.effective_permissions OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION create_owned_site(in_title character varying, in_subdomain_key character varying, in_theme_id bigint, in_user_id bigint, in_profile_name character varying, in_avatar_id bigint, in_avatar_url character varying, in_domain character varying DEFAULT NULL::character varying, in_description text DEFAULT NULL::text, in_logo_url character varying DEFAULT NULL::character varying, in_background_url character varying DEFAULT NULL::character varying, in_background_position character varying DEFAULT NULL::character varying, in_background_color character varying DEFAULT NULL::character varying, in_link_color character varying DEFAULT NULL::character varying, in_ga_web_property_id character varying DEFAULT NULL::character varying) RETURNS TABLE(new_site_id bigint, new_profile_id bigint)
    LANGUAGE plpgsql
    AS $$
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
$$;
-- +goose StatementEnd

ALTER FUNCTION development.create_owned_site(in_title character varying, in_subdomain_key character varying, in_theme_id bigint, in_user_id bigint, in_profile_name character varying, in_avatar_id bigint, in_avatar_url character varying, in_domain character varying, in_description text, in_logo_url character varying, in_background_url character varying, in_background_position character varying, in_background_color character varying, in_link_color character varying, in_ga_web_property_id character varying) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION get_communication_options(in_site_id integer, in_item_id integer, in_item_type_id integer, in_profile_id integer, in_update_type_id integer) RETURNS TABLE(send_email boolean, send_sms boolean)
    LANGUAGE plpgsql
    AS $$
BEGIN

IF (SELECT COUNT(*)
      FROM update_options
     WHERE profile_id = in_profile_id
       AND update_type_id = in_update_type_id) > 0 THEN

    RETURN QUERY
    SELECT o.send_email
          ,o.send_sms
      FROM update_options AS o
     WHERE o.profile_id = in_profile_id
       AND o.update_type_id = in_update_type_id;
ELSE

    DROP TABLE IF EXISTS tmp_identifiers;

    IF in_item_type_id >= 6 THEN --Item
        CREATE TEMP TABLE tmp_identifiers AS 
        SELECT in_profile_id AS profile_id
              ,microcosm_id
              ,site_id
              ,in_update_type_id AS update_type_id
          FROM flags
         WHERE item_id = in_item_id
           AND item_type_id = in_item_type_id;

    ELSIF in_item_type_id = 2 THEN --microcosm
        CREATE TEMP TABLE tmp_identifiers AS
        SELECT in_profile_id AS profile_id
              ,in_item_id AS microcosm_id
              ,m.site_id AS site_id
              ,in_update_type_id AS update_type_id
          FROM microcosms m
         WHERE m.microcosm_id = in_item_id;

    ELSIF in_item_type_id = 3 THEN --profile
        CREATE TEMP TABLE tmp_identifiers AS 
        SELECT in_profile_id AS profile_id
              ,0 AS microcosm_id
              ,p.site_id AS site_id
              ,in_update_type_id AS update_type_id
          FROM profiles p
         WHERE p.profile_id = in_item_id;

    ELSIF in_item_type_id = 4 THEN -- comment
        CREATE TEMP TABLE tmp_identifiers AS
        SELECT in_profile_id AS profile_id
              ,COALESCE(microcosm_id, 0) AS microcosm_id
              ,site_id
              ,in_update_type_id AS update_type_id
          FROM flags
         WHERE item_type_id = 4
           AND item_id = in_item_id;

    ELSE
        CREATE TEMP TABLE tmp_identifiers AS -- None of the above - either get defaults or 'send marketing notification'
        SELECT in_profile_id AS profile_id
              ,0 AS microcosm_id
              ,in_site_id AS site_id
              ,in_update_type_id AS update_type_id;

    END IF;

    IF (SELECT site_id FROM tmp_identifiers) <> in_site_id THEN
        RAISE EXCEPTION 'Inconsistent site id: received % and expected %', in_site_id, (SELECT site_id FROM tmp_identifiers);
    END IF;

    INSERT INTO update_options (
        profile_id, update_type_id, send_email, send_sms
    )
    SELECT in_profile_id as profile_id
          ,in_update_type_id AS update_type_id
          ,COALESCE(p.send_email, s.send_email, pl.send_email)
               AND (CASE in_update_type_id WHEN 0 THEN COALESCE(m.marketing_emails, TRUE) ELSE COALESCE(a.send_email, ad.send_email) END)
           AS send_email
          ,COALESCE(p.send_sms, s.send_sms, pl.send_sms)
               AND (CASE in_update_type_id WHEN 0 THEN COALESCE(m.marketing_sms, TRUE) ELSE COALESCE(a.send_sms, ad.send_sms) END)
           AS send_sms
      FROM tmp_identifiers i
           LEFT JOIN profile_options p ON p.profile_id = i.profile_id
           LEFT JOIN site_options s ON s.site_id = i.site_id
           LEFT JOIN update_options a ON a.profile_id = i.profile_id AND a.update_type_id = i.update_type_id
           LEFT JOIN update_options_defaults ad ON ad.update_type_id = i.update_type_id
           LEFT JOIN microcosm_profile_options m ON m.microcosm_id = i.microcosm_id AND m.profile_id = i.profile_id
           LEFT JOIN platform_options pl ON 1=1;

    RETURN QUERY
    SELECT o.send_email
          ,o.send_sms
      FROM update_options o
     WHERE o.profile_id = in_profile_id
       AND o.update_type_id = in_update_type_id;

END IF;
END

$$;
-- +goose StatementEnd

ALTER FUNCTION development.get_communication_options(in_site_id integer, in_item_id integer, in_item_type_id integer, in_profile_id integer, in_update_type_id integer) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION get_effective_permissions(insiteid bigint DEFAULT 0, inmicrocosmid bigint DEFAULT 0, initemtypeid bigint DEFAULT 0, initemid bigint DEFAULT 0, inprofileid bigint DEFAULT 0) RETURNS effective_permissions
    LANGUAGE plpgsql
    AS $_$
DECLARE
    result effective_permissions;
    mid bigint DEFAULT 0;
    parent_item_type_id bigint DEFAULT 0;
    parent_item_id bigint DEFAULT 0;
BEGIN
    -- Defaults
    SELECT false
          ,false
          ,false
          ,false
          ,false
          ,false
          ,false
          ,CASE WHEN (SELECT COUNT(*)
                        FROM profiles p
                       WHERE p.site_id = inSiteId
                         AND p.profile_id = inProfileId) = 0 THEN
                true ELSE false END
          ,false
          ,false
          ,false
          ,false
      INTO result.can_create
          ,result.can_read
          ,result.can_update
          ,result.can_delete
          ,result.can_close_own
          ,result.can_open_own
          ,result.can_read_others
          ,result.is_guest
          ,result.is_banned
          ,result.is_owner
          ,result.is_superuser
          ,result.is_site_owner;

    -- Are you banned? Defaults are good for you.
    IF (SELECT is_banned(inSiteId, inMicrocosmId, inProfileId)) THEN
        SELECT true
          INTO result.is_banned;

        -- banned people can still read the site
        IF inItemTypeId = 1 THEN
            SELECT true
              INTO result.can_read;
        END IF;

        -- banned people can still read their own profile
        IF inItemTypeId = 3 AND inItemId = inProfileId THEN
            SELECT true
              INTO result.can_read;
        END IF;

        RETURN result;
    END IF;

    -- Retrieve from cache if we are looking at a Microcosm and it is in cache
    IF (initemtypeid = 2 AND
        (SELECT COUNT(*) > 0
           FROM permissions_cache
          WHERE site_id = insiteid
            AND profile_id = inprofileid
            AND item_type_id = initemtypeid
            AND item_id = initemid) = True
    ) THEN
        SELECT can_create
              ,can_read
              ,can_update
              ,can_delete
              ,can_close_own
              ,can_open_own
              ,can_read_others
              ,is_guest
              ,is_banned
              ,is_owner
              ,is_superuser
              ,is_site_owner
          FROM permissions_cache
         WHERE site_id = insiteid
           AND profile_id = inprofileid
           AND item_type_id = initemtypeid
           AND item_id = initemid
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_guest
              ,result.is_owner
              ,result.is_owner
              ,result.is_superuser
              ,result.is_site_owner;

        RETURN result;
    END IF;

    -- What are you looking at
    mid := inMicrocosmId;

    IF ((inItemTypeId = 0 AND inItemId > 0) OR (mid = 0 AND inItemTypeId = 0 AND inItemId = 0)) THEN
        -- You are not looking at anything, this is nonsense, return defaults
        RETURN result;
    END IF;

    IF (mid > 0 AND inItemTypeId IN (1,3,4,5,14,16,17,18,19)) THEN
        -- Things that couldn't be in a Microcosm, so mid is irrelevant to us
        mid := 0;
    END IF;

    IF (mid = 0 AND inItemTypeId IN (2,4,6,7,9,13,15)) THEN
        -- Determine Microcosm ID from item
        CASE inItemTypeId
        WHEN 2 THEN -- Microcosm
            mid := inItemId;
        WHEN 4 THEN -- Comment
            SELECT COALESCE(f.microcosm_id, 0)
                  ,f.parent_item_type_id
                  ,f.parent_item_id
              FROM flags AS f
             WHERE f.item_type_id = 4
               AND f.item_id = inItemId
              INTO mid
                  ,parent_item_type_id
                  ,parent_item_id;
        WHEN 6 THEN -- Conversation
            SELECT co.microcosm_id
              FROM conversations co
             WHERE co.conversation_id = inItemId
              INTO mid;
        WHEN 7 THEN -- Poll
            SELECT p.microcosm_id
              FROM polls p
             WHERE p.poll_id = inItemId
              INTO mid;
        WHEN 9 THEN -- Event
            SELECT e.microcosm_id
              FROM events e
             WHERE e.event_id = inItemId
              INTO mid;
        WHEN 13 THEN -- Attendee
            SELECT i.microcosm_id
              FROM flags i,
                   attendees a
             WHERE a.attendee_id = inItemId
               AND i.item_type_id = 9 -- Event
               AND i.item_id = a.event_id
              INTO mid;
        WHEN 15 THEN -- Attribute
            SELECT i.microcosm_id
              FROM flags i
                  ,attribute_keys a
             WHERE i.item_type_id = a.item_type_id -- It could only exist in items if it existed in a Microcosm
               AND i.item_id = a.item_id
               AND a.attribute_id = inItemId
              INTO mid;

            IF mid IS NULL THEN
                mid := 0;
            END IF;
        END CASE;
    END IF;

    -- Are you banned? Defaults are good for you.
    IF inMicrocosmId = 0 AND mid > 0 AND (SELECT is_banned(inSiteId, mid, inProfileId)) THEN
        SELECT true
          INTO result.is_banned;

        RETURN result;
    END IF;

    -- Who are you
    IF $5 > 0 THEN
        -- Are you the site owner?
        SELECT (owned_by = inProfileId OR created_by = inProfileId)
              ,(owned_by = inProfileId OR created_by = inProfileId)
          FROM sites
         WHERE site_id = inSiteId
          INTO result.is_superuser
              ,result.is_site_owner;

        -- Are you a superuser?
        -- If site owner, then yes. If not, are you owner of the microcosm?
        IF result.is_superuser = false AND mid > 0 THEN
            -- Else look it up
            SELECT (owned_by = inProfileId OR created_by = inProfileId)
              FROM microcosms
             WHERE microcosm_id = mid
              INTO result.is_superuser;
        END IF;

        -- If you aren't the microcosm owner or site owner, do you have role identifying you as moderator?
        IF result.is_superuser = false AND mid > 0 THEN
            -- Else look it up
            IF (SELECT COUNT(*) FROM roles rr WHERE rr.microcosm_id = mid AND is_moderator_role = true) > 0 THEN
                -- microcosm roles
                SELECT (COUNT(*) > 0)
                 WHERE inProfileId IN (
                    SELECT get_role_profiles(1, role_id)
                      FROM roles
                     WHERE microcosm_id = mid
                       AND is_moderator_role = true
                       )
                  INTO result.is_superuser;
            ELSE
                -- default roles
                SELECT (COUNT(*) > 0)
                 WHERE inProfileId IN (
                    SELECT get_role_profiles(1, role_id)
                      FROM roles
                     WHERE microcosm_id IS NULL
                       AND is_moderator_role = true
                       )
                  INTO result.is_superuser;
            END IF;
        END IF;

        -- Do you own the item in question?
        IF (inItemTypeId > 0 AND inItemId > 0) THEN
            -- Determine owner from item
            -- Normalise attribute details first
            IF inItemTypeId = 15 THEN -- Attribute
                SELECT item_type_id
                      ,item_id
                  FROM attribute_keys
                 WHERE attribute_id = inItemId
                  INTO inItemTypeId
                      ,inItemId;
            END IF;

            CASE inItemTypeId
            WHEN 1 THEN -- Site
                SELECT (COUNT(*) > 0)
                  FROM sites
                 WHERE site_id = inItemId
                   AND (owned_by = inProfileId OR created_by = inProfileId)
                  INTO result.is_owner;

            WHEN 2 THEN -- Microcosm
                SELECT (COUNT(*) > 0)
                  FROM microcosms
                 WHERE microcosm_id = inItemId
                   AND (owned_by = inProfileId OR created_by = inProfileId)
                  INTO result.is_owner;

            WHEN 3 THEN -- Profile
                SELECT (COUNT(*) > 0)
                  FROM profiles
                 WHERE profile_id = inItemId
                   AND profile_id = inProfileId
                  INTO result.is_owner;

            WHEN 4 THEN -- Comment
                SELECT (COUNT(*) > 0)
                  FROM comments
                 WHERE comment_id = inItemId
                   AND profile_id = inProfileId
                  INTO result.is_owner;

            WHEN 5 THEN -- Huddle
                SELECT (COUNT(*) > 0)
                  FROM huddles
                 WHERE huddle_id = inItemId
                   AND created_by = inProfileId
                  INTO result.is_owner;

            WHEN 6 THEN -- Conversation
                SELECT (COUNT(*) > 0)
                  FROM conversations
                 WHERE conversation_id = inItemId
                   AND created_by = inProfileId
                  INTO result.is_owner;

            WHEN 7 THEN -- Poll
                SELECT (COUNT(*) > 0)
                  FROM polls
                 WHERE poll_id = inItemId
                   AND created_by = inProfileId
                  INTO result.is_owner;

            WHEN 8 THEN -- Article

            WHEN 9 THEN -- Event
                SELECT (COUNT(*) > 0)
                  FROM events
                 WHERE event_id = inItemId
                   AND created_by = inProfileId
                  INTO result.is_owner;

            WHEN 10 THEN -- Question

            WHEN 11 THEN -- Classified

            WHEN 12 THEN -- Album

            WHEN 13 THEN -- Attendee
                SELECT (COUNT(*) > 0)
                  FROM attendees
                 WHERE attendee_id = inItemId
                   AND (profile_id = inProfileId OR created_by = inProfileId)
                  INTO result.is_owner;

            WHEN 14 THEN -- User
                SELECT (COUNT(*) > 0)
                  FROM profiles
                 WHERE user_id = inItemId
                   AND site_id = inSiteId
                   AND profile_id = inProfileId
                  INTO result.is_owner;

            WHEN 15 THEN -- Attribute
                -- Isn't possible to do this, we figured this out
                -- just before this case statement

            WHEN 16 THEN -- Alert
                SELECT (COUNT(*) > 0)
                  FROM alerts
                 WHERE alert_id = inItemId
                   AND alerted_profile_id = inProfileId
                  INTO result.is_owner;

            WHEN 17 THEN -- Role
                SELECT (COUNT(*) > 0)
                  FROM roles
                 WHERE role_id = inItemId
                   AND created_by = inProfileId
                  INTO result.is_owner;

            WHEN 18 THEN -- AlertType
                -- Illogical

            WHEN 19 THEN -- Watcher
                SELECT (COUNT(*) > 0)
                  FROM watchers
                 WHERE watcher_id = inItemId
                   AND profile_id = inProfileId
                  INTO result.is_owner;

            ELSE
                -- Default, do nothing
            END CASE;
        END IF;
    END IF;

    -- What can you do to the thing you're looking at

    -- Permissions by type, things in a Microcosm look at the Microcosm
    -- permissions, whereas things owned by an individual look at that

    -- Create always refers to "something against this item"
    -- i.e. Create at Site, means Can Create Microcosm
    --      Create at Microcosm, means Can Create Items
    --      Create at Event, means Can Create Attendees
    -- etc
    CASE inItemTypeId
    WHEN 1 THEN -- Site
        SELECT (result.is_site_owner OR (so.only_admins_can_create_microcosms = false AND NOT (result.is_banned OR result.is_guest)))
              ,true -- Everyone can read the site info
              ,(result.is_site_owner OR result.is_owner)
              ,(result.is_site_owner)
          FROM site_options so
         WHERE so.site_id = insiteid
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete;

    WHEN 2 THEN -- Microcosm
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 3 THEN -- Profile
        SELECT (result.is_site_owner OR result.is_owner)
              ,true -- Everyone can read the profile info
              ,(result.is_site_owner OR result.is_owner)
              ,(result.is_site_owner)
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete;

    WHEN 4 THEN -- Comment
        IF mid > 0 THEN
            -- Fetch Microcosm permissions
            SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
                  ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
                  ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
                  ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
                  ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
                  ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
                  ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
              INTO result.can_create
                  ,result.can_read
                  ,result.can_update
                  ,result.can_delete
                  ,result.can_close_own
                  ,result.can_open_own
                  ,result.can_read_others;
         ELSE
             -- No mid means that this is a comment against a commentable type that doesn't
             -- exist in a Microcosm, such as a Huddle. As each type will have it's own rules
             -- we have to deal with those one at a time
             CASE parent_item_type_id
             WHEN 3 THEN -- Comment against a profile
                 SELECT (result.is_owner OR result.is_site_owner)
                       ,true
                       ,(result.is_owner OR result.is_site_owner)
                       ,(result.is_owner OR result.is_site_owner)
                       ,false
                       ,false
                       ,false
                   FROM (SELECT COUNT(*) AS in_huddle
                           FROM huddle_profiles
                          WHERE huddle_id = parent_item_id
                            AND profile_id = inProfileId
                        ) AS h
                   INTO result.can_create
                       ,result.can_read
                       ,result.can_update
                       ,result.can_delete
                       ,result.can_close_own
                       ,result.can_open_own
                       ,result.can_read_others;
             
             WHEN 5 THEN -- Comment against a huddle
                 SELECT (result.is_owner)
                       ,(h.in_huddle > 0 OR result.is_owner)
                       ,(result.is_owner)
                       ,false
                       ,false
                       ,false
                       ,false
                   FROM (SELECT COUNT(*) AS in_huddle
                           FROM huddle_profiles
                          WHERE huddle_id = parent_item_id
                            AND profile_id = inProfileId
                        ) AS h
                   INTO result.can_create
                       ,result.can_read
                       ,result.can_update
                       ,result.can_delete
                       ,result.can_close_own
                       ,result.can_open_own
                       ,result.can_read_others;
             ELSE
             END CASE;
         END IF;

    WHEN 5 THEN -- Huddle
        SELECT ((inItemId = 0 AND NOT result.is_guest) OR h.in_huddle > 0 OR result.is_owner)
              ,(h.in_huddle > 0 OR result.is_owner)
              ,(h.in_huddle > 0 OR result.is_owner)
              ,(h.in_huddle > 0 OR result.is_owner) -- delete means 'remove participant'
              ,false
              ,false
              ,false
          FROM (SELECT COUNT(*) AS in_huddle
                  FROM huddle_profiles
                 WHERE huddle_id = inItemId
                   AND profile_id = inProfileId
               ) AS h
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 6 THEN -- Conversation
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 7 THEN -- Poll
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 8 THEN -- Article
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 9 THEN -- Event
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 10 THEN -- Question
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 11 THEN -- Classified
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 12 THEN -- Album
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 13 THEN -- Attendee
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 14 THEN -- User
        SELECT (result.is_site_owner OR result.is_owner)
          INTO result.can_read;

    WHEN 15 THEN -- Attributes
        -- Already handled above, using the permissions of the thing the attribute is on

    WHEN 16 THEN -- Alert
        SELECT (result.is_owner)
              ,(result.is_owner)
              ,(result.is_owner)
          INTO result.can_read
              ,result.can_update
              ,result.can_delete;

    WHEN 17 THEN -- Role
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others;

    WHEN 18 THEN -- AlertType
        -- Illogical

    WHEN 19 THEN -- Watcher
        SELECT (result.is_site_owner OR result.is_owner)
              ,(result.is_site_owner OR result.is_owner)
              ,(result.is_site_owner OR result.is_owner)
              ,(result.is_site_owner OR result.is_owner)
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete;
    ELSE
        -- Illogical

    END CASE;

    IF inprofileid != 0 AND initemtypeid = 2 THEN
        INSERT INTO permissions_cache (
            site_id, profile_id, item_type_id, item_id, can_create,
            can_read, can_update, can_delete, can_close_own, can_open_own,
            can_read_others, is_guest, is_banned, is_owner, is_superuser,
            is_site_owner
        ) VALUES (
            insiteid, inprofileid, initemtypeid, initemid, result.can_create,
            result.can_read, result.can_update, result.can_delete, result.can_close_own, result.can_open_own,
            result.can_read_others, result.is_guest, result.is_banned, result.is_owner, result.is_superuser,
            result.is_site_owner
        );
    END IF;

    RETURN result;
END;
$_$;
-- +goose StatementEnd

ALTER FUNCTION development.get_effective_permissions(insiteid bigint, inmicrocosmid bigint, initemtypeid bigint, initemid bigint, inprofileid bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION get_microcosm_permissions_for_profile(insiteid bigint DEFAULT 0, inmicrocosmid bigint DEFAULT 0, inprofileid bigint DEFAULT 0) RETURNS effective_permissions
    LANGUAGE plpgsql STABLE
    AS $$
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
$$;
-- +goose StatementEnd

ALTER FUNCTION development.get_microcosm_permissions_for_profile(insiteid bigint, inmicrocosmid bigint, inprofileid bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION get_microcosm_roles(insiteid bigint DEFAULT 0, inmicrocosmid bigint DEFAULT 0) RETURNS SETOF bigint
    LANGUAGE plpgsql ROWS 1
    AS $$
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
$$;
-- +goose StatementEnd

ALTER FUNCTION development.get_microcosm_roles(insiteid bigint, inmicrocosmid bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION get_role_profile(insiteid bigint DEFAULT 0, inroleid bigint DEFAULT 0, inprofileid bigint DEFAULT 0) RETURNS SETOF bigint
    LANGUAGE plpgsql ROWS 1
    AS $$
DECLARE
    c role_members_cache%rowtype;
    hit boolean;
BEGIN
    hit := FALSE;

    FOR c IN
        SELECT site_id
              ,microcosm_id
              ,role_id
              ,profile_id
              ,in_role 
          FROM role_members_cache
         WHERE site_id = insiteid
           AND role_id = inroleid
           AND profile_id = inprofileid
    LOOP
        hit := TRUE;

        -- cache hit
        IF c.in_role = true THEN
            RETURN NEXT c.profile_id;
        END IF;
    END LOOP;
        
    IF hit = TRUE THEN
        RETURN;
    END IF;
    
    -- cache write
    SELECT site_id
          ,microcosm_id
          ,role_id
          ,inprofileid
          ,(
        SELECT COUNT(*)
          FROM (
            SELECT *
              FROM get_role_profiles(insiteid, inroleid)
               ) AS p
         WHERE p.get_role_profiles = inprofileid
           ) > 0
      FROM roles
     WHERE role_id = inroleid
      INTO c;

    INSERT INTO role_members_cache
    SELECT c.site_id
          ,c.microcosm_id
          ,c.role_id
          ,c.profile_id
          ,c.in_role
     WHERE NOT EXISTS (
        SELECT 1
          FROM role_members_cache
         WHERE site_id = c.site_id
           AND role_id = c.role_id
           AND profile_id = c.profile_id
               );

    IF c.in_role = TRUE THEN
        RETURN NEXT c.profile_id;
    END IF;

    RETURN;
END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.get_role_profile(insiteid bigint, inroleid bigint, inprofileid bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION get_role_profiles(site_id bigint DEFAULT 0, role_id bigint DEFAULT 0) RETURNS SETOF bigint
    LANGUAGE plpgsql STABLE
    AS $_$
DECLARE
    sql varchar;
    c criteria%rowtype;
    or_group bigint;
    count bigint;
BEGIN

    -- We can skip dynamic SQL if there are no criteria to process
    IF (
        (SELECT COUNT(*) = 0
          FROM criteria cr
         WHERE cr.role_id = $2) = True
       ) THEN

        -- Comment out for debugging
        --RAISE NOTICE 'Native query';

        RETURN QUERY
        SELECT 0
          FROM roles r
         WHERE r.role_id = $2
           AND r.include_guests = true
         UNION DISTINCT
        SELECT p.profile_id
          FROM roles r
               JOIN profiles p ON r.site_id = p.site_id
         WHERE r.role_id = $2
           AND r.include_users = true
         UNION DISTINCT
        SELECT rp.profile_id
          FROM role_profiles rp
         WHERE rp.role_id = $2;
    ELSE

    -- There must be some criteria, so dynamic SQL is the way to go

    -- guests
    SELECT CASE r.include_guests WHEN false THEN '' ELSE '
SELECT 0
 UNION DISTINCT' END
      FROM roles r
     WHERE r.role_id = $2
      INTO sql;

    -- users
    SELECT CASE r.include_users WHEN false THEN '' ELSE '
SELECT p.profile_id
  FROM profiles p
 WHERE p.site_id = $1
 UNION DISTINCT' END
      FROM roles r
     WHERE r.role_id = $2
      INTO sql;

    -- profiles explicitly assigned to role
    sql := sql || '
SELECT profile_id
  FROM role_profiles
 WHERE role_id = $2';

    -- profiles implicitly assigned to role via criteria
    count := 0;
    FOR c IN
        SELECT *
          FROM criteria
         WHERE criteria.role_id = $2
    LOOP
        IF count = 0 THEN
            or_group := c.or_group;
            sql := sql || '
 UNION DISTINCT
SELECT p.profile_id
  FROM profile_filter p
 WHERE p.site_id = $1
   AND (
       (';
        ELSE
            IF or_group = c.or_group THEN
                sql := sql || ' AND ';
            ELSE
                sql := sql || ')
    OR (';
            END IF;
        or_group := c.or_group;            
        END IF;

        IF c.profile_column IS NOT NULL THEN
            -- column on the profiles table
            CASE c.profile_column
            WHEN 'profile_name', 'email', 'gender' THEN
                -- strings
                CASE c.predicate
                WHEN 'eq' THEN
                    sql := sql || 'lower(p.' || c.profile_column || ') = lower(' || quote_literal(c.value) || ')';
                WHEN 'neq' THEN
                    sql := sql || 'lower(p.' || c.profile_column || ') != lower(' || quote_literal(c.value) || ')';
                WHEN 'substr' THEN
                    sql := sql || 'position(lower(' || quote_literal(c.value) || ') in lower(p.' || c.profile_column || ')) > 0';
                WHEN 'nsubstr' THEN
                    sql := sql || 'position(lower(' || quote_literal(c.value) || ') in lower(p.' || c.profile_column || ')) = 0';
                ELSE
                    sql := sql || '11=11';
                END CASE;
            WHEN 'profile_id', 'item_count', 'comment_count' THEN
                -- numbers
                CASE c.predicate
                WHEN 'eq' THEN
                    sql := sql || 'p.' || c.profile_column || ' = ' || cast(c.value as int);
                WHEN 'neq' THEN
                    sql := sql || 'p.' || c.profile_column || ' != ' || cast(c.value as int);
                WHEN 'lt' THEN
                    sql := sql || 'p.' || c.profile_column || ' < ' || cast(c.value as int);
                WHEN 'le' THEN
                    sql := sql || 'p.' || c.profile_column || ' <= ' || cast(c.value as int);
                WHEN 'ge' THEN
                    sql := sql || 'p.' || c.profile_column || ' >= ' || cast(c.value as int);
                WHEN 'gt' THEN
                    sql := sql || 'p.' || c.profile_column || ' > ' || cast(c.value as int);
                ELSE
                    sql := sql || '12=12';
                END CASE;
            WHEN 'created' THEN
                -- dates
                CASE c.predicate
                WHEN 'eq' THEN
                    sql := sql || 'EXTRACT(DAY FROM NOW() - p.' || c.profile_column || ') = ' || cast(c.value as int);
                WHEN 'neq' THEN
                    sql := sql || 'EXTRACT(DAY FROM NOW() - p.' || c.profile_column || ') != ' || cast(c.value as int);
                WHEN 'lt' THEN
                    sql := sql || 'EXTRACT(DAY FROM NOW() - p.' || c.profile_column || ') < ' || cast(c.value as int);
                WHEN 'le' THEN
                    sql := sql || 'EXTRACT(DAY FROM NOW() - p.' || c.profile_column || ') <= ' || cast(c.value as int);
                WHEN 'ge' THEN
                    sql := sql || 'EXTRACT(DAY FROM NOW() - p.' || c.profile_column || ') >= ' || cast(c.value as int);
                WHEN 'gt' THEN
                    sql := sql || 'EXTRACT(DAY FROM NOW() - p.' || c.profile_column || ') > ' || cast(c.value as int);
                ELSE
                    sql := sql || '13=13';
                END CASE;
            ELSE
                sql:= sql || '10=10';
            END CASE;
        ELSIF c.key IS NOT NULL THEN
            -- value within the attribute_keys and attribute_values tables
            CASE c.type
            WHEN 'string' THEN
                --strings
                CASE c.predicate
                WHEN 'eq' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 1 --string
   AND v.string = ' || quote_literal(c.value) || '
)';
                WHEN 'ne' THEN
                    sql := sql || 'p.profile_id IN (
SELECT profile_id
  FROM profiles
 WHERE profile_id NOT IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 1 --string
   AND v.string = ' || quote_literal(c.value) || '
))';
                WHEN 'substr' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 1 --string
   AND position(lower(' || quote_literal(c.value) || ') in lower(v.string)) > 0
)';
                WHEN 'nsubstr' THEN
                    sql := sql || 'p.profile_id IN (
SELECT profile_id
  FROM profiles
 WHERE profile_id NOT IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 1 --string
   AND position(lower(' || quote_literal(c.value) || ') in lower(v.string)) > 0
))';
                ELSE
                    sql := sql || '21=21';
                END CASE;
            WHEN 'number' THEN
                --numbers
                CASE c.predicate
                WHEN 'eq' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 3 --number
   AND v.number = ' || cast(c.value as numeric) || '
)';
                WHEN 'neq' THEN
                    sql := sql || 'p.profile_id IN (
SELECT profile_id
  FROM profiles
 WHERE profile_id NOT IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 3 --number
   AND v.number = ' || cast(c.value as numeric) || '
))';
                WHEN 'lt' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 3 --number
   AND v.number < ' || cast(c.value as numeric) || '
)';
                WHEN 'le' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 3 --number
   AND v.number <= ' || cast(c.value as numeric) || '
)';
                WHEN 'ge' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 3 --number
   AND v.number >= ' || cast(c.value as numeric) || '
)';
                WHEN 'gt' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 3 --number
   AND v.number > ' || cast(c.value as numeric) || '
)';
                ELSE
                    sql := sql || '22=22';
                END CASE;
            WHEN 'boolean' THEN
                CASE c.predicate
                WHEN 'eq' THEN
                    sql := sql || 'p.profile_id IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 4 --boolean
   AND v.boolean = ' || cast(c.value as boolean) || '
)';
                WHEN 'neq' THEN
                    sql := sql || 'p.profile_id IN (
SELECT profile_id
  FROM profiles
 WHERE profile_id NOT IN (
SELECT k.item_id
  FROM attribute_keys k, attribute_values v
 WHERE k.item_type_id = 3 --profile
   AND k.key = ' || quote_literal(c.key) || '
   AND k.attribute_id = v.attribute_id
   AND v.value_type_id = 4 --boolean
   AND v.boolean = ' || cast(c.value as boolean) || '
))';
                ELSE
                    sql := sql || '23=23';
                END CASE;
            ELSE
                sql := sql || '20=20';
            END CASE;
        END IF;

        count := count + 1;
    END LOOP;

    IF count > 0 THEN
        sql := sql || ')
       )';
    END IF;

    sql := sql || '
 ORDER BY 1 ASC';

    -- Comment out for debugging
    --RAISE NOTICE 'Made some SQL: %', sql;

    RETURN QUERY EXECUTE sql
    USING $1, $2;

    END IF;
END;
$_$;
-- +goose StatementEnd

ALTER FUNCTION development.get_role_profiles(site_id bigint, role_id bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION has_unread(in_item_type_id bigint DEFAULT 0, in_item_id bigint DEFAULT 0, in_profile_id bigint DEFAULT 0) RETURNS boolean
    LANGUAGE plpgsql STABLE
    AS $$
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
$$;
-- +goose StatementEnd

ALTER FUNCTION development.has_unread(in_item_type_id bigint, in_item_id bigint, in_profile_id bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION is_attending(in_event_id bigint, in_profile_id bigint) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN

RETURN COUNT(*)>0 as is_attending
FROM attendees
WHERE event_id = in_event_id
AND profile_id = in_profile_id
AND state_id=1;

END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.is_attending(in_event_id bigint, in_profile_id bigint) OWNER TO microcosm;

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

ALTER FUNCTION development.is_banned(site_id bigint, microcosm_id bigint, profile_id bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION is_deleted(initemtypeid bigint DEFAULT 0, initemid bigint DEFAULT 0) RETURNS boolean
    LANGUAGE plpgsql STABLE
    AS $$
DECLARE
BEGIN

    IF inItemTypeId = 0 THEN
        RETURN true;
    END IF;

    CASE inItemTypeId
    WHEN 1 THEN -- Site
        RETURN (
            SELECT COUNT(site_id) = 0 AS is_deleted
              FROM sites
             WHERE site_id = inItemId
        );

    WHEN 2 THEN -- Microcosm
        RETURN (
            SELECT COUNT(m.microcosm_id) = 0 AS is_deleted
              FROM microcosms m
             WHERE m.microcosm_id = inItemId
               AND NOT m.is_deleted
               --AND is_deleted(CAST(1 AS BIGINT), m.site_id) = false
        );

    WHEN 3 THEN -- Profile
        RETURN (
            SELECT COUNT(p.profile_id) = 0 AS is_deleted
              FROM profiles p
             WHERE p.profile_id = inItemId
               --AND is_deleted(CAST(1 AS BIGINT), p.site_id) = false
        );

    WHEN 4 THEN -- Comment
        RETURN (
            SELECT COUNT(d.is_deleted) = 0 AS is_deleted
              FROM (             
                    SELECT FALSE AS is_deleted
                      FROM flags
                     WHERE item_type_id = 4
                       AND item_id = inItemId
                       AND NOT item_is_deleted
                       AND NOT item_is_moderated
                       AND NOT parent_is_deleted
                       AND NOT parent_is_moderated
                       AND (
                               parent_item_type_id = 5
                            OR (
                                   NOT microcosm_is_deleted
                               AND NOT microcosm_is_moderated
                               )
                           )
                   ) AS d
        );

    WHEN 5 THEN -- Huddle
        RETURN (
            SELECT COUNT(h.huddle_id) = 0 AS is_deleted
              FROM huddles h
             WHERE h.huddle_id = inItemId
               --AND is_deleted(CAST(1 AS BIGINT), h.site_id) = false
        );

    WHEN 6 THEN -- Conversation
        RETURN (
            SELECT COUNT(c.conversation_id) = 0 AS is_deleted
              FROM conversations c
                   JOIN microcosms m ON m.microcosm_id = c.microcosm_id
             WHERE c.conversation_id = inItemId
               AND NOT c.is_deleted
               AND NOT m.is_deleted
               --AND is_deleted(CAST(1 AS BIGINT), m.site_id) = false
        );

    WHEN 7 THEN -- Poll
        RETURN (
            SELECT COUNT(p.poll_id) = 0 AS is_deleted
              FROM polls p
                   JOIN microcosms m ON m.microcosm_id = p.microcosm_id
             WHERE p.poll_id = inItemId
               AND NOT p.is_deleted
               AND NOT m.is_deleted
               --AND is_deleted(CAST(1 AS BIGINT), m.site_id) = false
        );

    WHEN 8 THEN -- Article

    WHEN 9 THEN -- Event
        RETURN (
            SELECT COUNT(e.event_id) = 0 AS is_deleted
              FROM events e
                   JOIN microcosms m ON m.microcosm_id = e.microcosm_id
             WHERE e.event_id = inItemId
               AND NOT e.is_deleted
               AND NOT m.is_deleted
               --AND is_deleted(CAST(1 AS BIGINT), m.site_id) = false
        );

    WHEN 10 THEN -- Question

    WHEN 11 THEN -- Classified

    WHEN 12 THEN -- Album

    WHEN 13 THEN -- Attendee
        RETURN (
            SELECT COUNT(attendee_id) = 0 AS is_deleted
              FROM attendees a
                   JOIN events e ON e.event_id = a.event_id
                   JOIN microcosms m ON m.microcosm_id = e.microcosm_id
             WHERE a.attendee_id = inItemId
               AND NOT e.is_deleted
               AND NOT m.is_deleted
        );

    WHEN 14 THEN -- User
        RETURN (
            SELECT COUNT(user_id) = 0 AS is_deleted
              FROM users u
             WHERE u.user_id = inItemId
        );

    WHEN 15 THEN -- Attribute
        RETURN (
            SELECT COUNT(a.attribute_id) = 0 AS is_deleted
              FROM attribute_keys a
             WHERE a.attribute_id = inItemId
               AND NOT is_deleted(a.item_type_id, a.item_id)
        );

    WHEN 16 THEN -- Update
        RETURN (
            SELECT COUNT(u.update_id) = 0 AS is_deleted
              FROM updates u
             WHERE u.update_id = inItemId
               AND NOT is_deleted(u.item_type_id, u.item_id)
               --AND is_deleted(CAST(1 AS BIGINT), u.site_id) = false
        );

    WHEN 17 THEN -- Role

    WHEN 18 THEN -- UpdateType

    WHEN 19 THEN -- Watcher

    END CASE;

    RETURN TRUE;

END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.is_deleted(initemtypeid bigint, initemid bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION last_read_time(in_item_type_id bigint DEFAULT 0, in_item_id bigint DEFAULT 0, in_profile_id bigint DEFAULT 0) RETURNS timestamp without time zone
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN

    IF in_profile_id = 0 THEN
        RETURN '1970-01-01 00:00:01.000'::date AS last_read;
    END IF;

    CASE in_item_type_id
    WHEN 1 THEN -- site
    WHEN 2 THEN -- microcosm
    WHEN 3 THEN -- profile
    WHEN 4 THEN -- comment
    WHEN 5 THEN -- huddle

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags i
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 6 THEN -- conversation

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 2 AND item_id = ( -- microcosm last read
                     SELECT microcosm_id 
                       FROM flags 
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 7 THEN -- poll

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 2 AND item_id = ( -- microcosm last read
                     SELECT microcosm_id 
                       FROM flags 
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 8 THEN -- article

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 2 AND item_id = ( -- microcosm last read
                     SELECT microcosm_id 
                       FROM flags 
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 9 THEN -- event

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 2 AND item_id = ( -- microcosm last read
                     SELECT microcosm_id 
                       FROM flags 
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 10 THEN -- question

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 2 AND item_id = ( -- microcosm last read
                     SELECT microcosm_id 
                       FROM flags 
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 11 THEN -- classified

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 2 AND item_id = ( -- microcosm last read
                     SELECT microcosm_id 
                       FROM flags 
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 12 THEN -- album

        RETURN COALESCE(MAX(read), '1970-01-01 00:00:01.000'::date) AS last_read
          FROM read
         WHERE (
                   -- item last read
                   (item_type_id = in_item_type_id AND item_id = in_item_id)
                OR (item_type_id = 2 AND item_id = ( -- microcosm last read
                     SELECT microcosm_id 
                       FROM flags 
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
                OR (item_type_id = 1 AND item_id = ( -- site last read
                     SELECT site_id
                       FROM flags
                      WHERE item_type_id = in_item_type_id
                        AND item_id = in_item_id
                       )
                   )
               )
           AND profile_id = in_profile_id;

    WHEN 13 THEN -- attendee
    WHEN 14 THEN -- user
    WHEN 15 THEN -- attribute
    WHEN 16 THEN -- update
    WHEN 17 THEN -- role
    WHEN 18 THEN -- update type
    WHEN 19 THEN -- watcher
    END CASE;

    RETURN '1970-01-01 00:00:01.000'::date AS last_read;

END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.last_read_time(in_item_type_id bigint, in_item_id bigint, in_profile_id bigint) OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_comments_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 4
               AND item_id = OLD.comment_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.is_deleted <> OLD.is_deleted OR
               NEW.is_moderated <> OLD.is_moderated OR
               NEW.item_type_id <> OLD.item_type_id OR
               NEW.item_id <> OLD.item_id THEN
       
            IF NEW.item_type_id = 5 THEN
                UPDATE flags
                   SET item_is_deleted = NEW.is_deleted
                      ,item_is_moderated = NEW.is_moderated
                      ,microcosm_id = NULL
                      ,microcosm_is_deleted = false
                      ,microcosm_is_moderated = false
                      ,parent_item_type_id = NEW.item_type_id
                      ,parent_item_id = NEW.item_id
                      ,parent_is_deleted = false
                      ,parent_is_moderated = false
                 WHERE item_type_id = 4
                   AND item_id = NEW.comment_id;
            ELSE
                UPDATE flags AS f
                   SET item_is_deleted = NEW.is_deleted
                      ,item_is_moderated = NEW.is_moderated
                      ,microcosm_id = i.microcosm_id
                      ,microcosm_is_deleted = i.microcosm_is_deleted
                      ,microcosm_is_moderated = i.microcosm_is_moderated
                      ,parent_item_type_id = NEW.item_type_id
                      ,parent_item_id = NEW.item_id
                      ,parent_is_deleted = i.item_is_deleted
                      ,parent_is_moderated = i.item_is_moderated
                  FROM flags AS i
                 WHERE i.parent_item_type_id = NEW.item_type_id
                   AND i.parent_item_id = NEW.item_id
                   AND f.item_type_id = 4
                   AND f.item_id = NEW.comment_id;
            END IF;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            IF NEW.item_type_id = 5 THEN

                INSERT INTO flags (
                    site_id
                   ,item_type_id
                   ,item_id
                   ,item_is_deleted
                   ,item_is_moderated
                   ,parent_item_type_id
                   ,parent_item_id
                   ,created_by
                )
                SELECT h.site_id
                      ,4
                      ,NEW.comment_id
                      ,NEW.is_deleted
                      ,NEW.is_moderated
                      ,NEW.item_type_id
                      ,NEW.item_id
                      ,NEW.profile_id
                  FROM huddles h
                 WHERE huddle_id = NEW.item_id;

                 UPDATE flags
                    SET last_modified = NEW.created
                  WHERE item_type_id = NEW.item_type_id
                    AND item_id = NEW.item_id;

            ELSE

                INSERT INTO flags (
                    site_id
                   ,microcosm_id
                   ,microcosm_is_moderated
                   ,microcosm_is_deleted
                   ,item_type_id

                   ,item_id
                   ,item_is_deleted
                   ,item_is_moderated
                   ,parent_item_type_id
                   ,parent_item_id

                   ,parent_is_deleted
                   ,parent_is_moderated
                   ,last_modified
                   ,created_by
                )
                SELECT i.site_id
                      ,i.microcosm_id
                      ,i.microcosm_is_moderated
                      ,i.microcosm_is_deleted
                      ,4

                      ,NEW.comment_id
                      ,NEW.is_deleted
                      ,NEW.is_moderated
                      ,NEW.item_type_id
                      ,NEW.item_id

                      ,i.item_is_deleted
                      ,i.item_is_moderated
                      ,NEW.created
                      ,NEW.profile_id
                  FROM flags AS i
                 WHERE i.item_type_id = NEW.item_type_id
                   AND i.item_id = NEW.item_id;

                 UPDATE flags
                    SET last_modified = NEW.created
                  WHERE item_type_id = NEW.item_type_id
                    AND item_id = NEW.item_id;

                 UPDATE flags
                    SET last_modified = NEW.created
                   FROM (
                            SELECT microcosm_id
                              FROM flags
                             WHERE item_type_id = NEW.item_type_id
                               AND item_id = NEW.item_id
                        ) AS p
                  WHERE item_type_id = 2
                    AND item_id = p.microcosm_id;

            END IF;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_comments_flags() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_comments_search_index() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 4
               AND item_id = OLD.comment_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.item_type_id <> OLD.item_type_id OR
               NEW.item_id <> OLD.item_id THEN
        
            UPDATE search_index
               SET parent_item_type_id = NEW.item_type_id
                  ,parent_item_id = NEW.item_id
             WHERE item_type_id = 4
               AND item_id = NEW.comment_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            -- handled by revisions.INSERT

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_comments_search_index() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_conversations_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 6
               AND item_id = OLD.conversation_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.is_deleted <> OLD.is_deleted OR
               NEW.is_moderated <> OLD.is_moderated OR
               NEW.is_sticky <> OLD.is_sticky OR
               NEW.microcosm_id <> OLD.microcosm_id THEN

            -- Item
            UPDATE flags AS f
               SET microcosm_is_deleted = m.is_deleted
                  ,microcosm_is_moderated = m.is_moderated
                  ,item_is_deleted = NEW.is_deleted
                  ,item_is_moderated = NEW.is_moderated
                  ,item_is_sticky = NEW.is_sticky
                  ,microcosm_id = NEW.microcosm_id
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id
               AND f.item_type_id = 6
               AND f.item_id = NEW.conversation_id;

            -- Children (comments)
            UPDATE flags
               SET microcosm_is_deleted = m.is_deleted
                  ,microcosm_is_moderated = m.is_moderated
                  ,parent_is_deleted = NEW.is_deleted
                  ,parent_is_moderated = NEW.is_moderated
                  ,microcosm_id = NEW.microcosm_id
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id
               AND parent_item_type_id = 6
               AND parent_item_id = NEW.conversation_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,microcosm_id
               ,microcosm_is_deleted
               ,microcosm_is_moderated
               ,item_type_id
               ,item_id
               ,item_is_deleted
               ,item_is_moderated
               ,item_is_sticky
               ,last_modified
               ,created_by
            )
            SELECT m.site_id
                  ,NEW.microcosm_id
                  ,m.is_deleted
                  ,m.is_moderated
                  ,6
                  ,NEW.conversation_id
                  ,NEW.is_deleted
                  ,NEW.is_moderated
                  ,NEW.is_sticky
                  ,NEW.created
                  ,NEW.created_by
              FROM microcosms AS m
             WHERE m.microcosm_id = NEW.microcosm_id;

            UPDATE flags
               SET last_modified = NEW.created
             WHERE item_type_id = 2
               AND item_id = NEW.microcosm_id;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_conversations_flags() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_conversations_search_index() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 6
               AND item_id = OLD.conversation_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.microcosm_id <> OLD.microcosm_id OR
               NEW.title <> OLD.title THEN
        
            UPDATE search_index
               SET microcosm_id = NEW.microcosm_id
                  ,title_text = NEW.title
                  ,title_vector = setweight(to_tsvector(NEW.title), 'C')
                  ,document_text = NEW.title
                  ,document_vector = setweight(to_tsvector(NEW.title), 'C')
                  ,last_modified = NOW()
             WHERE item_type_id = 6
               AND item_id = NEW.conversation_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO search_index (
                site_id
               ,microcosm_id
               ,profile_id
               ,item_type_id
               ,item_id

               ,title_text
               ,title_vector
               ,document_text
               ,document_vector
               ,last_modified
            )
            SELECT m.site_id
                  ,NEW.microcosm_id
                  ,NEW.created_by
                  ,6
                  ,NEW.conversation_id
                  ,NEW.title

                  ,setweight(to_tsvector(NEW.title), 'C')
                  ,NEW.title
                  ,setweight(to_tsvector(NEW.title), 'C')
                  ,NOW()
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_conversations_search_index() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_events_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 9
               AND item_id = OLD.event_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.is_deleted <> OLD.is_deleted OR
               NEW.is_moderated <> OLD.is_moderated OR
               NEW.is_sticky <> OLD.is_sticky OR
               NEW.microcosm_id <> OLD.microcosm_id THEN

            -- Item
            UPDATE flags AS f
               SET microcosm_is_deleted = m.is_deleted
                  ,microcosm_is_moderated = m.is_moderated
                  ,item_is_deleted = NEW.is_deleted
                  ,item_is_moderated = NEW.is_moderated
                  ,item_is_sticky = NEW.is_sticky
                  ,microcosm_id = NEW.microcosm_id
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id
               AND f.item_type_id = 9
               AND f.item_id = NEW.event_id;

            -- Children (comments)
            UPDATE flags
               SET microcosm_is_deleted = m.is_deleted
                  ,microcosm_is_moderated = m.is_moderated
                  ,parent_is_deleted = NEW.is_deleted
                  ,parent_is_moderated = NEW.is_moderated
                  ,microcosm_id = NEW.microcosm_id
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id
               AND parent_item_type_id = 9
               AND parent_item_id = NEW.event_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,microcosm_id
               ,microcosm_is_deleted
               ,microcosm_is_moderated
               ,item_type_id
               ,item_id
               ,item_is_deleted
               ,item_is_moderated
               ,item_is_sticky
               ,last_modified
               ,created_by
            )
            SELECT m.site_id
                  ,NEW.microcosm_id
                  ,m.is_deleted
                  ,m.is_moderated
                  ,9
                  ,NEW.event_id
                  ,NEW.is_deleted
                  ,NEW.is_moderated
                  ,NEW.is_sticky
                  ,NEW.created
                  ,NEW.created_by
              FROM microcosms AS m
             WHERE m.microcosm_id = NEW.microcosm_id;

            UPDATE flags
               SET last_modified = NEW.created
             WHERE item_type_id = 2
               AND item_id = NEW.microcosm_id;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_events_flags() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_events_search_index() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 9
               AND item_id = OLD.event_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.microcosm_id <> OLD.microcosm_id OR
               NEW.title <> OLD.title THEN

            UPDATE search_index
               SET microcosm_id = NEW.microcosm_id
                  ,title_text = NEW.title
                  ,title_vector = setweight(to_tsvector(NEW.title), 'C')
                  ,document_text = NEW.title || ' ' || COALESCE(NEW.where, '')
                  ,document_vector = setweight(to_tsvector(NEW.title), 'C') || setweight(to_tsvector(COALESCE(NEW.where, '')), 'D')
                  ,last_modified = NOW()
             WHERE item_type_id = 9
               AND item_id = NEW.event_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO search_index (
                site_id
               ,microcosm_id
               ,profile_id
               ,item_type_id
               ,item_id

               ,title_text
               ,title_vector
               ,document_text
               ,document_vector
               ,last_modified
            )
            SELECT m.site_id
                  ,m.microcosm_id
                  ,NEW.created_by
                  ,9
                  ,NEW.event_id

                  ,NEW.title
                  ,setweight(to_tsvector(NEW.title), 'C')
                  ,NEW.title || ' ' || COALESCE(NEW.where, '')
                  ,setweight(to_tsvector(NEW.title), 'C') || setweight(to_tsvector(COALESCE(NEW.where, '')), 'D')
                  ,NEW.created
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_events_search_index() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_huddles_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 5
               AND item_id = OLD.huddle_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN
            -- We have nothing meaningful to do here for a huddle
            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,item_type_id
               ,item_id
               ,last_modified
               ,created_by
            )
            SELECT NEW.site_id
                  ,5
                  ,NEW.huddle_id
                  ,NEW.created
                  ,NEW.created_by;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_huddles_flags() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_huddles_search_index() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 5
               AND item_id = OLD.huddle_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.title <> OLD.title THEN

            UPDATE search_index
               SET title_text = NEW.title
                  ,title_vector = setweight(to_tsvector(NEW.title), 'C')
                  ,document_text = NEW.title
                  ,document_vector = setweight(to_tsvector(NEW.title), 'C')
                  ,last_modified = NOW()
             WHERE item_type_id = 5
               AND item_id = NEW.huddle_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO search_index (
                site_id
               ,profile_id
               ,item_type_id
               ,item_id
               ,title_text

               ,title_vector
               ,document_text
               ,document_vector
               ,last_modified
            )
            SELECT NEW.site_id
                  ,NEW.created_by
                  ,5
                  ,NEW.huddle_id
                  ,NEW.title

                  ,setweight(to_tsvector(NEW.title), 'C')
                  ,NEW.title
                  ,setweight(to_tsvector(NEW.title), 'C')
                  ,NEW.created;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_huddles_search_index() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_microcosms_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 2
               AND item_id = OLD.microcosm_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.is_deleted <> OLD.is_deleted OR
               NEW.is_moderated <> OLD.is_moderated OR
               NEW.is_sticky <> OLD.is_sticky THEN

            -- Microcosms
            UPDATE flags
               SET item_is_deleted = NEW.is_deleted
                  ,item_is_moderated = NEW.is_moderated
                  ,item_is_sticky = NEW.is_sticky
                  ,last_modified = COALESCE(NEW.edited, NOW())
             WHERE item_type_id = 2
               AND item_id = NEW.microcosm_id;

            -- Children (items)
            UPDATE flags
               SET microcosm_is_deleted = NEW.is_deleted
                  ,microcosm_is_moderated = NEW.is_moderated
             WHERE microcosm_id = NEW.microcosm_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,item_type_id
               ,item_id
               ,item_is_deleted
               ,item_is_moderated
               ,item_is_sticky
               ,last_modified
               ,created_by
            )
            SELECT NEW.site_id
                  ,2
                  ,NEW.microcosm_id
                  ,NEW.is_deleted
                  ,NEW.is_moderated
                  ,NEW.is_sticky
                  ,NEW.created
                  ,NEW.created_by;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_microcosms_flags() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_microcosms_search_index() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 2
               AND item_id = OLD.microcosm_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN
            IF NEW.title <> OLD.title OR NEW.description <> OLD.description THEN
        
                UPDATE search_index
                   SET title_text = NEW.title
                      ,title_vector = setweight(to_tsvector(NEW.title), 'A')
                      ,document_text = NEW.title || ' ' || NEW.description
                      ,document_vector = setweight(to_tsvector(NEW.title), 'A') || setweight(to_tsvector(NEW.description), 'D')
                      ,last_modified = NOW()
                 WHERE item_type_id = 2
                   AND item_id = NEW.microcosm_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO search_index (
                site_id
               ,profile_id
               ,item_type_id
               ,item_id
               ,title_text

               ,title_vector
               ,document_text
               ,document_vector
               ,last_modified
            ) VALUES (
                NEW.site_id
               ,NEW.created_by
               ,2
               ,NEW.microcosm_id
               ,NEW.title

               ,setweight(to_tsvector(NEW.title), 'A')
               ,NEW.title || ' ' || NEW.description
               ,setweight(to_tsvector(NEW.title), 'A') || setweight(to_tsvector(NEW.description), 'D')
               ,NEW.created
            );

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_microcosms_search_index() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_polls_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 7
               AND item_id = OLD.poll_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.is_deleted <> OLD.is_deleted OR
               NEW.is_moderated <> OLD.is_moderated OR
               NEW.is_sticky <> OLD.is_sticky OR
               NEW.microcosm_id <> OLD.microcosm_id THEN

            -- Item
            UPDATE flags AS f
               SET microcosm_is_deleted = m.is_deleted
                  ,microcosm_is_moderated = m.is_moderated
                  ,item_is_deleted = NEW.is_deleted
                  ,item_is_moderated = NEW.is_moderated
                  ,item_is_sticky = NEW.is_sticky
                  ,m.microcosm_id = NEW.microcosm_id
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id
               AND f.item_type_id = 7
               AND f.item_id = NEW.poll_id;

            -- Children (comments)
            UPDATE flags
               SET microcosm_is_deleted = m.is_deleted
                  ,microcosm_is_moderated = m.is_moderated
                  ,parent_is_deleted = NEW.is_deleted
                  ,parent_is_moderated = NEW.is_moderated
                  ,m.microcosm_id = NEW.microcosm_id
              FROM microcosms m
             WHERE m.microcosm_id = NEW.microcosm_id
               AND parent_item_type_id = 7
               AND parent_item_id = NEW.poll_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO flags (
                site_id
               ,microcosm_id
               ,microcosm_is_deleted
               ,microcosm_is_moderated
               ,item_type_id
               ,item_id
               ,item_is_deleted
               ,item_is_moderated
               ,item_is_sticky
               ,last_modified
               ,created_by
            )
            SELECT m.site_id
                  ,NEW.microcosm_id
                  ,m.is_deleted
                  ,m.is_moderated
                  ,7
                  ,NEW.poll_id
                  ,NEW.is_deleted
                  ,NEW.is_moderated
                  ,NEW.is_sticky
                  ,NEW.created
                  ,NEW.created_by
              FROM microcosms AS m
             WHERE m.microcosm_id = NEW.microcosm_id;

            UPDATE flags
               SET last_modified = NEW.created
             WHERE item_type_id = 2
               AND item_id = NEW.microcosm_id;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_polls_flags() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_profiles_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 3
               AND item_id = OLD.profile_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN
            -- There is nothing meaningful we can currently do here
            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,item_type_id
               ,item_id
               ,last_modified
               ,created_by
            )
            SELECT NEW.site_id
                  ,3
                  ,NEW.profile_id
                  ,NEW.created
                  ,NEW.profile_id;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_profiles_flags() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_profiles_search_index() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 3
               AND item_id = OLD.profile_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.profile_name <> OLD.profile_name THEN

            UPDATE search_index
               SET title_text = NEW.profile_name
                  ,title_vector = setweight(to_tsvector(NEW.profile_name), 'B')
                  ,document_text = NEW.profile_name
                  ,document_vector = setweight(to_tsvector(NEW.profile_name), 'B')
                  ,last_modified = NOW()
             WHERE item_type_id = 3
               AND item_id = NEW.profile_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO search_index (
                site_id
               ,profile_id
               ,item_type_id
               ,item_id
               ,title_text

               ,title_vector
               ,document_text
               ,document_vector
               ,last_modified
            )
            SELECT NEW.site_id
                  ,NEW.profile_id
                  ,3
                  ,NEW.profile_id
                  ,NEW.profile_name

                  ,setweight(to_tsvector(NEW.profile_name), 'B')
                  ,NEW.profile_name
                  ,setweight(to_tsvector(NEW.profile_name), 'B')
                  ,NEW.created;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_profiles_search_index() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_revisions_search_index() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') AND OLD.is_current THEN
            DELETE
              FROM search_index
             WHERE item_type_id = 4
               AND item_id = OLD.comment_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN

            IF NEW.raw <> OLD.raw THEN

            UPDATE search_index
               SET document_text = NEW.raw
                  ,document_vector = setweight(to_tsvector(NEW.raw), 'D')
                  ,last_modified = NOW()
             WHERE item_type_id = 4
               AND item_id = NEW.comment_id;

            END IF;

            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            IF (
                SELECT COUNT(*) = 0 
                  FROM search_index
                 WHERE item_type_id = 4
                   AND item_id = NEW.comment_id
            ) THEN

                INSERT INTO search_index (
                    site_id
                   ,profile_id
                   ,item_type_id
                   ,item_id
                   ,parent_item_type_id

                   ,parent_item_id
                   ,document_text
                   ,document_vector
                   ,last_modified
                )
                SELECT p.site_id
                      ,p.profile_id
                      ,4
                      ,NEW.comment_id
                      ,c.item_type_id

                      ,c.item_id
                      ,NEW.raw
                      ,setweight(to_tsvector(NEW.raw), 'D')
                      ,NEW.created
                  FROM profiles p
                      ,"comments" c
                 WHERE p.profile_id = NEW.profile_id
                   AND c.comment_id = NEW.comment_id;

            ELSE

                UPDATE search_index
                   SET document_text = NEW.raw
                      ,document_vector = setweight(to_tsvector(NEW.raw), 'D')
                      ,last_modified = NOW()
                 WHERE item_type_id = 4
                   AND item_id = NEW.comment_id;

            END IF;

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_revisions_search_index() OWNER TO microcosm;

-- +goose StatementBegin
CREATE FUNCTION update_sites_flags() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (TG_OP = 'DELETE') THEN

            DELETE
              FROM flags
             WHERE item_type_id = 1
               AND item_id = OLD.site_id;

            RETURN OLD;

        ELSIF (TG_OP = 'UPDATE') THEN
            -- We have nothing meaningful to do here for a site
            RETURN NEW;

        ELSIF (TG_OP = 'INSERT') THEN

            INSERT INTO flags (
                site_id
               ,item_type_id
               ,item_id
               ,last_modified
               ,created_by
            )
            SELECT NEW.site_id
                  ,1
                  ,NEW.site_id
                  ,NEW.created
                  ,(SELECT MIN(profile_id) FROM profiles LIMIT 1);

            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$$;
-- +goose StatementEnd

ALTER FUNCTION development.update_sites_flags() OWNER TO microcosm;

SET default_tablespace = '';

SET default_with_oids = false;

CREATE TABLE access_tokens (
    access_token_id bigint NOT NULL,
    token_value character varying(128) NOT NULL,
    user_id bigint NOT NULL,
    client_id bigint NOT NULL,
    created timestamp without time zone DEFAULT now() NOT NULL,
    expires timestamp without time zone DEFAULT (now() + '1 year'::interval) NOT NULL
);

ALTER TABLE development.access_tokens OWNER TO microcosm;

CREATE SEQUENCE access_tokens_access_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.access_tokens_access_token_id_seq OWNER TO microcosm;

ALTER SEQUENCE access_tokens_access_token_id_seq OWNED BY access_tokens.access_token_id;

CREATE TABLE activity_scores (
    site_id bigint NOT NULL,
    item_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    score bigint NOT NULL,
    updated timestamp without time zone DEFAULT now() NOT NULL
);

ALTER TABLE development.activity_scores OWNER TO microcosm;

COMMENT ON TABLE activity_scores IS 'Stores an integer score for any kind of item. The scoring algorithm and update frequency is determined by the application.';

CREATE TABLE admins (
    admin_id bigint NOT NULL,
    site_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    created timestamp without time zone NOT NULL
);

ALTER TABLE development.admins OWNER TO microcosm;

COMMENT ON TABLE admins IS 'Simple list of which profiles have admin rights on a given sites.

Admins are the people who effectively own and manage a given site.';

CREATE TABLE updates (
    update_id bigint NOT NULL,
    for_profile_id bigint NOT NULL,
    update_type_id bigint NOT NULL,
    item_type_id bigint,
    item_id bigint,
    created_by bigint,
    created timestamp without time zone NOT NULL,
    site_id bigint,
    parent_item_type_id bigint DEFAULT 0 NOT NULL,
    parent_item_id bigint DEFAULT 0 NOT NULL
);

ALTER TABLE development.updates OWNER TO microcosm;

COMMENT ON TABLE updates IS 'List of updates that have been sent to people.

In Google Plus this is the drop down box on the top right telling you what has happened since you last logged on, but by storing more than just the last 8 we can also show you your prior updates too which helps when you''ve been on vacation and come back, and overwhelmed you just want to go filter down to "just those who replied to comments I''ve made".';

COMMENT ON COLUMN updates.item_id IS 'item_id is *NOT* to be foreign keyed to any particular table.

Depending on item_type_id this column contains a value relating to one of the tables implied by that.

That is to say, if item_type_id states that the item type is a comment then the item_id is a value that is a valid comments.comment_id.

And if the item_type_id states that the item type is a conversation then the item_id is a value that is a valid conversations.conversation_id.';

CREATE SEQUENCE alerts_alert_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.alerts_alert_id_seq OWNER TO microcosm;

ALTER SEQUENCE alerts_alert_id_seq OWNED BY updates.update_id;

CREATE TABLE attachment_meta (
    attachment_meta_id bigint NOT NULL,
    created timestamp without time zone NOT NULL,
    file_size integer NOT NULL,
    width integer,
    height integer,
    thumbnail_width integer,
    thumbnail_height integer,
    attach_count integer DEFAULT 1 NOT NULL,
    file_sha1 character varying(40) NOT NULL,
    mime_type character varying(255) NOT NULL,
    file_name character varying(512) DEFAULT 'untitled.unk'::character varying NOT NULL,
    file_ext character varying(255) DEFAULT 'unk'::character varying NOT NULL
);

ALTER TABLE development.attachment_meta OWNER TO microcosm;

COMMENT ON TABLE attachment_meta IS 'Describes an actual file, stored somewhere.';

CREATE SEQUENCE attachment_meta_attachment_meta_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.attachment_meta_attachment_meta_id_seq OWNER TO microcosm;

ALTER SEQUENCE attachment_meta_attachment_meta_id_seq OWNED BY attachment_meta.attachment_meta_id;

CREATE TABLE attachment_views (
    attachment_id bigint NOT NULL
);

ALTER TABLE development.attachment_views OWNER TO microcosm;

COMMENT ON TABLE attachment_views IS 'Here just as a performance thing, any time someone views an attachment, put an entry in here.

A scheduled task should COUNT(*) GROUP BY on this table to update the attachments table view_count.

The purpose is to delay update to attachments, and reduce the number of updates. It''s to prevent locking the table too much.';

CREATE TABLE attachments (
    attachment_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    attachment_meta_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    created timestamp without time zone NOT NULL,
    view_count integer DEFAULT 0 NOT NULL,
    file_sha1 character varying(40) NOT NULL,
    file_name character varying(512) DEFAULT 'untitled.unk'::character varying NOT NULL,
    file_ext character varying(255) DEFAULT 'unk'::character varying NOT NULL
);

ALTER TABLE development.attachments OWNER TO microcosm;

COMMENT ON TABLE attachments IS 'Takes an attachment (described in attachment_meta) and associates it (attaches it to) some item or post.';

COMMENT ON COLUMN attachments.item_id IS 'item_id is *NOT* to be foreign keyed to any particular table.

Depending on item_type_id this column contains a value relating to one of the tables implied by that.

That is to say, if item_type_id states that the item type is a comment then the item_id is a value that is a valid comments.comment_id.

And if the item_type_id states that the item type is a conversation then the item_id is a value that is a valid conversations.conversation_id.';

CREATE SEQUENCE attachments_attachment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.attachments_attachment_id_seq OWNER TO microcosm;

ALTER SEQUENCE attachments_attachment_id_seq OWNED BY attachments.attachment_id;

CREATE TABLE attendee_state (
    state_id bigint NOT NULL,
    title character varying(10) NOT NULL
);

ALTER TABLE development.attendee_state OWNER TO microcosm;

CREATE TABLE attendees (
    attendee_id bigint NOT NULL,
    event_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    created timestamp without time zone NOT NULL,
    created_by bigint NOT NULL,
    edited timestamp without time zone,
    edited_by bigint,
    edit_reason character varying(150),
    state_id bigint NOT NULL,
    state_date timestamp without time zone NOT NULL
);

ALTER TABLE development.attendees OWNER TO microcosm;

COMMENT ON COLUMN attendees.profile_id IS 'In the case of attendees, this is the person attending the event';

COMMENT ON COLUMN attendees.created_by IS 'If profile_id != created_by then this is the person inviting the other person to the event';

CREATE SEQUENCE attendees_attendee_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.attendees_attendee_id_seq OWNER TO microcosm;

ALTER SEQUENCE attendees_attendee_id_seq OWNED BY attendees.attendee_id;

CREATE TABLE attribute_keys (
    attribute_id bigint NOT NULL,
    item_type_id bigint,
    item_id bigint,
    key character varying(50)
);

ALTER TABLE development.attribute_keys OWNER TO microcosm;

COMMENT ON TABLE attribute_keys IS 'Describes attributes that exist on any given entity. Values for attributes can be found in the values table.

Note: Attributes keys must be unique for a given item. That is, you cannot two attributes that share the same key but different datatypes.';

CREATE TABLE attribute_values (
    attribute_id bigint NOT NULL,
    value_type_id bigint,
    string text,
    date date,
    number numeric,
    "boolean" boolean,
    CONSTRAINT values_check CHECK (((((string IS NOT NULL) OR (date IS NOT NULL)) OR (number IS NOT NULL)) OR ("boolean" IS NOT NULL)))
);

ALTER TABLE development.attribute_values OWNER TO microcosm;

COMMENT ON TABLE attribute_values IS 'Stores the values for attributes.

Aside from attribute_id, one (and only one) of the fields IS NOT NULL, all of the other fields are NULL.

Note: If this table has fewer than 8 columns then NULLs take up zero space.';

CREATE SEQUENCE attributes_attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.attributes_attribute_id_seq OWNER TO microcosm;

ALTER SEQUENCE attributes_attribute_id_seq OWNED BY attribute_keys.attribute_id;

CREATE TABLE banned_emails (
    site_id bigint NOT NULL,
    user_id bigint NOT NULL,
    email character varying(254) NOT NULL
);

ALTER TABLE development.banned_emails OWNER TO microcosm;

COMMENT ON TABLE banned_emails IS 'Stores banned emails, these should be with some understanding that + and . in email addresses have special meaning. This is part of troll and spam fighting.';

CREATE TABLE bans (
    ban_id bigint NOT NULL,
    site_id bigint,
    user_id bigint NOT NULL,
    created timestamp without time zone NOT NULL,
    expires timestamp without time zone,
    display_reason text,
    admin_reason text
);

ALTER TABLE development.bans OWNER TO microcosm;

COMMENT ON TABLE bans IS 'Bans are temporary (or permanent) exiles from a given site. I am split as to whether a microcosm should be able to ban someone, inclined to say yes... but at the moment these are site level bans.

Reasons for bans are stored twice, once as a polite message to the banned person, and once as an admin/moderator audit trail of the root cause.';

CREATE SEQUENCE bans_ban_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.bans_ban_id_seq OWNER TO microcosm;

ALTER SEQUENCE bans_ban_id_seq OWNED BY bans.ban_id;

CREATE TABLE choices (
    choice_id bigint NOT NULL,
    poll_id bigint NOT NULL,
    title character varying(150) NOT NULL,
    sequence integer DEFAULT 0 NOT NULL,
    vote_count integer DEFAULT 0 NOT NULL,
    voter_count integer DEFAULT 0 NOT NULL
);

ALTER TABLE development.choices OWNER TO microcosm;

COMMENT ON TABLE choices IS 'Choices for polls. As in, the things you vote for on a poll.';

CREATE SEQUENCE choices_choice_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.choices_choice_id_seq OWNER TO microcosm;

ALTER SEQUENCE choices_choice_id_seq OWNED BY choices.choice_id;

CREATE TABLE comments (
    comment_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    created timestamp without time zone NOT NULL,
    in_reply_to bigint,
    is_visible boolean DEFAULT true NOT NULL,
    is_moderated boolean DEFAULT false NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    attachment_count integer DEFAULT 0 NOT NULL,
    yay_count integer DEFAULT 0 NOT NULL,
    meh_count integer DEFAULT 0 NOT NULL,
    grr_count integer DEFAULT 0 NOT NULL
);

ALTER TABLE development.comments OWNER TO microcosm;

COMMENT ON TABLE comments IS 'Comments are the heart of a community, these are the posts in a traditional forum. The actual content is stored in revisions.';

COMMENT ON COLUMN comments.item_id IS 'item_id is *NOT* to be foreign keyed to any particular table.

Depending on item_type_id this column contains a value relating to one of the tables implied by that.

That is to say, if item_type_id states that the item type is a comment then the item_id is a value that is a valid comments.comment_id.

And if the item_type_id states that the item type is a conversation then the item_id is a value that is a valid conversations.conversation_id.';

CREATE SEQUENCE comments_comment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.comments_comment_id_seq OWNER TO microcosm;

ALTER SEQUENCE comments_comment_id_seq OWNED BY comments.comment_id;

CREATE TABLE conversations (
    conversation_id bigint NOT NULL,
    microcosm_id bigint NOT NULL,
    title character varying(150) NOT NULL,
    created timestamp without time zone NOT NULL,
    created_by bigint NOT NULL,
    edited timestamp without time zone,
    edited_by bigint,
    edit_reason character varying(150),
    is_sticky boolean DEFAULT false NOT NULL,
    is_open boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    is_moderated boolean DEFAULT false NOT NULL,
    is_visible boolean DEFAULT true NOT NULL,
    comment_count integer DEFAULT 0 NOT NULL,
    view_count integer DEFAULT 0 NOT NULL
);

ALTER TABLE development.conversations OWNER TO microcosm;

COMMENT ON TABLE conversations IS 'Conversations are a chronologically sequenced collection of comments.

These are threads in most forum software.';

CREATE SEQUENCE conversations_conversation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.conversations_conversation_id_seq OWNER TO microcosm;

ALTER SEQUENCE conversations_conversation_id_seq OWNED BY conversations.conversation_id;

CREATE TABLE criteria (
    criteria_id bigint NOT NULL,
    role_id bigint,
    or_group bigint DEFAULT 0 NOT NULL,
    profile_column character varying(50),
    key character varying(50),
    type character varying(10),
    predicate character varying(10) NOT NULL,
    value character varying(250) NOT NULL,
    CONSTRAINT criteria_check1 CHECK (((profile_column IS NOT NULL) OR (key IS NOT NULL))),
    CONSTRAINT criteria_predicate_check CHECK (((predicate)::text = ANY (ARRAY[('eq'::character varying)::text, ('ne'::character varying)::text, ('lt'::character varying)::text, ('le'::character varying)::text, ('ge'::character varying)::text, ('gt'::character varying)::text, ('substr'::character varying)::text, ('nsubstr'::character varying)::text]))),
    CONSTRAINT criteria_type_check CHECK (((type IS NULL) OR ((type)::text = ANY (ARRAY[('string'::character varying)::text, ('number'::character varying)::text, ('boolean'::character varying)::text, ('date'::character varying)::text]))))
);

ALTER TABLE development.criteria OWNER TO microcosm;

COMMENT ON TABLE criteria IS 'Defines the criteria by which membership of a role is determined. Basically the rows within this are used to create a SQL statement that selects users, such that they are implicitly included as members of a role.';

COMMENT ON COLUMN criteria.criteria_id IS 'Criteria ID is pretty cosmetic and just to assist REST interfaces to ensure that there is some identifier to this rule.';

COMMENT ON COLUMN criteria.role_id IS 'The role to which this criteria is attached.';

COMMENT ON COLUMN criteria.or_group IS 'An integer identifier for a group of rules.

Any criteria sharing the same or_group value and role_id is grouped by AND.

Any criteria with a distinct or_group value and sharing the same role_id is grouped by OR.

i.e. If you have 3 criteria and they have 2 distinct or_group values:

{
  {criteria_id: 1, or_group: 1},
  {criteria_id: 2, or_group: 2},
  {criteria_id: 3, or_group: 2}
}

The this would create SQL conceptually grouped thus:

WHERE (criteria: 1) OR (criteria:2 AND criteria:3)

Default is 0, so unless specified criteria are AND statements.';

COMMENT ON COLUMN criteria.profile_column IS 'If not null, then this is the name of the column on the profiles table to which the criteria is applied.';

COMMENT ON COLUMN criteria.key IS 'If not null, this corresponds to the attributes table and indicates which key contains the values the critera applies to.';

COMMENT ON COLUMN criteria.type IS 'If key is not null, then this is the datatype of the value of key.

One of:

string
number
boolean
date

Notes: date values do not include times, and number represents all types of numbers.';

COMMENT ON COLUMN criteria.predicate IS 'Encapsulates the rule for matching rows.

All types
eq = Equals
ne = Not equals

Numbers and Dates only
lt = Less than
le = Less than or equals
ge = Greater than or equals
gt = Greater than

Strings only
substr = Value is in string
nsubstr = Value is not in string';

COMMENT ON COLUMN criteria.value IS 'The value to which the predicate must return true for the specified column or property value.';

CREATE SEQUENCE criteria_criteria_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.criteria_criteria_id_seq OWNER TO microcosm;

ALTER SEQUENCE criteria_criteria_id_seq OWNED BY criteria.criteria_id;

CREATE TABLE disabled_roles (
    microcosm_id bigint NOT NULL,
    role_id bigint NOT NULL
);

ALTER TABLE development.disabled_roles OWNER TO microcosm;

COMMENT ON TABLE disabled_roles IS 'If a row exists in here, than any role explicitly or implicitly (default roles) assigned to a Microcosm are marked as disabled.';

CREATE TABLE events (
    event_id bigint NOT NULL,
    microcosm_id bigint NOT NULL,
    title character varying(150) NOT NULL,
    created timestamp without time zone NOT NULL,
    created_by bigint NOT NULL,
    edited timestamp without time zone,
    edited_by bigint,
    edit_reason character varying(150),
    is_sticky boolean DEFAULT false NOT NULL,
    is_open boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    is_moderated boolean DEFAULT false NOT NULL,
    is_visible boolean DEFAULT true NOT NULL,
    comment_count integer DEFAULT 0 NOT NULL,
    view_count integer DEFAULT 0 NOT NULL,
    "when" timestamp without time zone,
    duration integer DEFAULT 0 NOT NULL,
    "where" character varying(150),
    lat double precision,
    lon double precision,
    bounds_north double precision,
    bounds_east double precision,
    bounds_south double precision,
    bounds_west double precision,
    status character varying(20) DEFAULT 'proposed'::character varying NOT NULL,
    rsvp_limit integer DEFAULT 0 NOT NULL,
    rsvp_attending integer DEFAULT 0 NOT NULL,
    rsvp_spaces integer NOT NULL,
    is_full boolean DEFAULT false NOT NULL,
    CONSTRAINT status_check CHECK (((status)::text = ANY (ARRAY[('proposed'::character varying)::text, ('upcoming'::character varying)::text, ('postponed'::character varying)::text, ('cancelled'::character varying)::text, ('past'::character varying)::text])))
);

ALTER TABLE development.events OWNER TO microcosm;

COMMENT ON COLUMN events.is_full IS 'Indicates whether the event is full, i.e. rsvp_attending == rsvp_spaces';

CREATE SEQUENCE events_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.events_event_id_seq OWNER TO microcosm;

ALTER SEQUENCE events_event_id_seq OWNED BY events.event_id;

CREATE TABLE flags (
    site_id bigint NOT NULL,
    site_is_deleted boolean DEFAULT false NOT NULL,
    microcosm_id bigint,
    microcosm_is_deleted boolean DEFAULT false NOT NULL,
    microcosm_is_moderated boolean DEFAULT false NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    item_is_deleted boolean DEFAULT false NOT NULL,
    item_is_moderated boolean DEFAULT false NOT NULL,
    parent_item_type_id bigint,
    parent_item_id bigint,
    parent_is_deleted boolean DEFAULT false NOT NULL,
    parent_is_moderated boolean DEFAULT false NOT NULL,
    item_is_sticky boolean DEFAULT false NOT NULL,
    last_modified timestamp without time zone DEFAULT now() NOT NULL,
    created_by bigint DEFAULT 0,
    sitemap_file bigint,
    sitemap_index bigint DEFAULT 0 NOT NULL
);

ALTER TABLE development.flags OWNER TO microcosm;

CREATE TABLE follows (
    profile_id bigint NOT NULL,
    follow_profile_id bigint NOT NULL,
    created timestamp without time zone NOT NULL
);

ALTER TABLE development.follows OWNER TO microcosm;

COMMENT ON TABLE follows IS 'Almost called this the stalker table. Follows are asymmetric pointers between profiles. Unsure of how this will surface, but the idea is twofold: 1) that you can change your privacy option to limit things to followers (benefit to them), and that 2) you can see what the people you are following have been up to (benefit to you).';

CREATE TABLE huddle_profiles (
    huddle_id bigint NOT NULL,
    profile_id bigint NOT NULL
);

ALTER TABLE development.huddle_profiles OWNER TO microcosm;

COMMENT ON TABLE huddle_profiles IS 'For a given huddle, lists the people who are privy to it.';

CREATE TABLE huddles (
    huddle_id bigint NOT NULL,
    site_id bigint NOT NULL,
    title character varying DEFAULT 150 NOT NULL,
    created timestamp without time zone NOT NULL,
    created_by bigint NOT NULL,
    is_confidential boolean DEFAULT false NOT NULL
);

ALTER TABLE development.huddles OWNER TO microcosm;

COMMENT ON TABLE huddles IS 'Huddles are private conversations that people can join (if included by an existing huddle member) or leave. Uses comments to store the actual data, but this stores the idea of the huddle.

Huddles are basically private conversations.

When no rows exist in huddle_profiles, then the huddle and all corresponding comments should be physically deleted.';

CREATE SEQUENCE huddles_huddle_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.huddles_huddle_id_seq OWNER TO microcosm;

ALTER SEQUENCE huddles_huddle_id_seq OWNED BY huddles.huddle_id;

CREATE TABLE ignores (
    profile_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL
);

ALTER TABLE development.ignores OWNER TO microcosm;

CREATE TABLE import_origins (
    origin_id bigint NOT NULL,
    title character varying(150) NOT NULL,
    site_id bigint NOT NULL,
    product character varying,
    major_version integer DEFAULT 0 NOT NULL,
    minor_version integer DEFAULT 0 NOT NULL,
    imported_comments_threaded boolean DEFAULT false NOT NULL,
    imported_follows boolean DEFAULT false NOT NULL
);

ALTER TABLE development.import_origins OWNER TO microcosm;

COMMENT ON TABLE import_origins IS 'This is the site/domain/dump from which items are to be imported.

An import should only ever happen once, and it should be possible to resume a broken import.';

COMMENT ON COLUMN import_origins.product IS 'Stores the product that the data came from, for example ''vbulletin''';

CREATE SEQUENCE import_origins_origin_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.import_origins_origin_id_seq OWNER TO microcosm;

ALTER SEQUENCE import_origins_origin_id_seq OWNED BY import_origins.origin_id;

CREATE TABLE imported_items (
    origin_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    old_id character varying(64) NOT NULL,
    item_id bigint NOT NULL
);

ALTER TABLE development.imported_items OWNER TO microcosm;

COMMENT ON TABLE imported_items IS 'The items imported.

We map the old id to the new for a couple of reasons:

1) To allow an import to break, and for us to skip already imported items.

2) To allow for URL rewrite patterns that could redirect old URLs to the new URLs.

For the latter reason, we will not purge this table after a successful import.';

COMMENT ON COLUMN imported_items.item_id IS 'item_id is *NOT* to be foreign keyed to any particular table.

Depending on item_type_id this column contains a value relating to one of the tables implied by that.

That is to say, if item_type_id states that the item type is a comment then the item_id is a value that is a valid comments.comment_id.

And if the item_type_id states that the item type is a conversation then the item_id is a value that is a valid conversations.conversation_id.';

CREATE TABLE ips (
    site_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    seen timestamp without time zone DEFAULT now() NOT NULL,
    action "char" NOT NULL,
    ip inet NOT NULL
);

ALTER TABLE development.ips OWNER TO microcosm;

CREATE TABLE item_types (
    item_type_id bigint NOT NULL,
    title character varying(32) NOT NULL,
    rank smallint DEFAULT 0 NOT NULL
);

ALTER TABLE development.item_types OWNER TO microcosm;

COMMENT ON TABLE item_types IS 'Declares all types that can be searched for by the global search bar.

Acts as a disambiguator for items within a microcosm.';

COMMENT ON COLUMN item_types.rank IS 'Search weighting for this type of item... a higher number is a greater weight.';

CREATE TABLE links (
    link_id bigint NOT NULL,
    short_url character varying(250) NOT NULL,
    domain character varying(1024) NOT NULL,
    url character varying(8096) NOT NULL,
    inner_text character varying(8096) NOT NULL,
    created timestamp without time zone DEFAULT now() NOT NULL,
    resolved_url character varying(8096),
    resolved timestamp without time zone,
    hits bigint DEFAULT 0
);

ALTER TABLE development.links OWNER TO microcosm;

COMMENT ON TABLE links IS 'Hyperlinks within comments';

CREATE SEQUENCE links_link_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.links_link_id_seq OWNER TO microcosm;

ALTER SEQUENCE links_link_id_seq OWNED BY links.link_id;

CREATE TABLE menus (
    menu_id bigint NOT NULL,
    site_id bigint NOT NULL,
    href character varying(2000) NOT NULL,
    title character varying(512),
    text character varying(50) NOT NULL,
    sequence smallint DEFAULT 0 NOT NULL,
    CONSTRAINT menus_sequence_check CHECK ((sequence <= 10)),
    CONSTRAINT menus_sequence_check1 CHECK ((sequence >= 0))
);

ALTER TABLE development.menus OWNER TO microcosm;

COMMENT ON TABLE menus IS 'Allows for 10 menu items to be created on the navigation of a site.

The navigation menu should be named after the site itself (site.title), and the items in the menu come from this table and should be sorted by sequence.';

COMMENT ON COLUMN menus.menu_id IS 'Identifies a menu item';

COMMENT ON COLUMN menus.href IS 'The link destination, can either both on-site or off-site.';

COMMENT ON COLUMN menus.title IS 'Optional. If the text for the anchor is not clear, the title provides an accessible way to provide additional information about the link destination.';

COMMENT ON COLUMN menus.text IS 'The title of the link, as it will appear between the anchor tags.';

COMMENT ON COLUMN menus.sequence IS 'Menu items should be sorted by this field, it is a required field and must also be unique. Values can only be 0-10.';

CREATE SEQUENCE menus_menu_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.menus_menu_id_seq OWNER TO microcosm;

ALTER SEQUENCE menus_menu_id_seq OWNED BY menus.menu_id;

CREATE TABLE metrics (
    job_timestamp timestamp without time zone NOT NULL,
    visits integer DEFAULT 0 NOT NULL,
    uniques integer DEFAULT 0 NOT NULL,
    new_profiles integer DEFAULT 0 NOT NULL,
    edited_profiles integer DEFAULT 0 NOT NULL,
    total_profiles integer DEFAULT 0 NOT NULL,
    signins integer DEFAULT 0 NOT NULL,
    comments integer DEFAULT 0 NOT NULL,
    conversations integer DEFAULT 0 NOT NULL,
    engaged_forums integer DEFAULT 0 NOT NULL,
    total_forums integer DEFAULT 0 NOT NULL,
    pageviews integer DEFAULT 0 NOT NULL
);

ALTER TABLE development.metrics OWNER TO microcosm;

CREATE TABLE microcosm_options (
    microcosm_id bigint NOT NULL
);

ALTER TABLE development.microcosm_options OWNER TO microcosm;

CREATE TABLE microcosm_profile_options (
    microcosm_profile_option_id integer NOT NULL,
    microcosm_id bigint,
    profile_id bigint,
    marketing_emails boolean,
    marketing_alerts boolean,
    marketing_sms boolean
);

ALTER TABLE development.microcosm_profile_options OWNER TO microcosm;

CREATE SEQUENCE microcosm_profile_options_microcosm_profile_option_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.microcosm_profile_options_microcosm_profile_option_id_seq OWNER TO microcosm;

ALTER SEQUENCE microcosm_profile_options_microcosm_profile_option_id_seq OWNED BY microcosm_profile_options.microcosm_profile_option_id;

CREATE TABLE microcosms (
    microcosm_id bigint NOT NULL,
    title character varying(50) NOT NULL,
    description text,
    site_id bigint NOT NULL,
    visibility character varying(10) DEFAULT 'public'::character varying NOT NULL,
    created timestamp without time zone NOT NULL,
    created_by bigint NOT NULL,
    edited timestamp without time zone,
    edited_by bigint,
    edit_reason character varying(150),
    owned_by bigint NOT NULL,
    style_id bigint,
    item_count integer DEFAULT 0 NOT NULL,
    comment_count integer DEFAULT 0 NOT NULL,
    last_comment_id bigint,
    last_comment_created timestamp without time zone,
    last_comment_created_by bigint,
    last_comment_item_type_id bigint,
    last_comment_item_id bigint,
    last_comment_item_title character varying(50),
    is_sticky boolean DEFAULT false NOT NULL,
    is_moderated boolean DEFAULT false NOT NULL,
    is_open boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    is_visible boolean DEFAULT false NOT NULL,
    CONSTRAINT visibility_check CHECK (((visibility)::text = ANY (ARRAY[('public'::character varying)::text, ('private'::character varying)::text, ('protected'::character varying)::text])))
);

ALTER TABLE development.microcosms OWNER TO microcosm;

COMMENT ON TABLE microcosms IS 'Microcosms are the buckets, ''forums'' in traditional software. They are buckets of items around a topic.';

CREATE SEQUENCE microcosms_microcosm_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.microcosms_microcosm_id_seq OWNER TO microcosm;

ALTER SEQUENCE microcosms_microcosm_id_seq OWNED BY microcosms.microcosm_id;

CREATE TABLE moderation_queue (
    moderation_queue_id bigint NOT NULL,
    microcosm_id bigint,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    created timestamp without time zone NOT NULL
);

ALTER TABLE development.moderation_queue OWNER TO microcosm;

COMMENT ON TABLE moderation_queue IS 'Each microcosm has a moderation queue.

All moderators for a microcosm, as well as the admins of the site that owns the microcosm and those of sites who host the microcosm have the ability to moderate microcosm content.

Microcosms are not moderated by default, but if they are, then items will appear in this queue prior to be displayed in the microcosm.';

COMMENT ON COLUMN moderation_queue.item_id IS 'item_id is *NOT* to be foreign keyed to any particular table.

Depending on item_type_id this column contains a value relating to one of the tables implied by that.

That is to say, if item_type_id states that the item type is a comment then the item_id is a value that is a valid comments.comment_id.

And if the item_type_id states that the item type is a conversation then the item_id is a value that is a valid conversations.conversation_id.';

CREATE SEQUENCE moderation_queue_moderation_queue_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.moderation_queue_moderation_queue_id_seq OWNER TO microcosm;

ALTER SEQUENCE moderation_queue_moderation_queue_id_seq OWNED BY moderation_queue.moderation_queue_id;

CREATE TABLE oauth_clients (
    client_id bigint NOT NULL,
    name character varying(80) NOT NULL,
    created timestamp without time zone DEFAULT now() NOT NULL,
    client_secret character varying NOT NULL
);

ALTER TABLE development.oauth_clients OWNER TO microcosm;

COMMENT ON TABLE oauth_clients IS 'Stores oauth2 client applications (e.g. an Android app that requires access to a user''s account).';

CREATE SEQUENCE oauth_clients_client_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.oauth_clients_client_id_seq OWNER TO microcosm;

ALTER SEQUENCE oauth_clients_client_id_seq OWNED BY oauth_clients.client_id;

CREATE TABLE permissions_cache (
    site_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    can_create boolean NOT NULL,
    can_read boolean NOT NULL,
    can_update boolean NOT NULL,
    can_delete boolean NOT NULL,
    can_close_own boolean NOT NULL,
    can_open_own boolean NOT NULL,
    can_read_others boolean NOT NULL,
    is_guest boolean NOT NULL,
    is_banned boolean NOT NULL,
    is_owner boolean NOT NULL,
    is_superuser boolean NOT NULL,
    is_site_owner boolean NOT NULL
);

ALTER TABLE development.permissions_cache OWNER TO microcosm;

CREATE TABLE platform_options (
    send_email boolean NOT NULL,
    send_sms boolean NOT NULL
);

ALTER TABLE development.platform_options OWNER TO microcosm;

CREATE TABLE polls (
    poll_id bigint NOT NULL,
    microcosm_id bigint NOT NULL,
    title character varying(150) NOT NULL,
    question character varying(512) NOT NULL,
    created timestamp without time zone NOT NULL,
    created_by bigint NOT NULL,
    edited timestamp without time zone,
    edited_by bigint,
    edit_reason character varying(150),
    voter_count integer DEFAULT 0 NOT NULL,
    voting_ends timestamp without time zone,
    is_sticky boolean DEFAULT false NOT NULL,
    is_open boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    is_moderated boolean DEFAULT false NOT NULL,
    is_visible boolean DEFAULT true NOT NULL,
    comment_count integer DEFAULT 0 NOT NULL,
    view_count integer DEFAULT 0 NOT NULL,
    is_poll_open boolean DEFAULT false NOT NULL,
    is_multiple_choice boolean DEFAULT false NOT NULL
);

ALTER TABLE development.polls OWNER TO microcosm;

COMMENT ON TABLE polls IS 'Single-choice or multiple-choice polls, optionally with chronologically ordered comments.

Polls can be open to the public (anon users), in which case the voter_count and vote_count columns would be incremented without corresponding rows in the votes table.';

CREATE SEQUENCE polls_poll_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.polls_poll_id_seq OWNER TO microcosm;

ALTER SEQUENCE polls_poll_id_seq OWNED BY polls.poll_id;

CREATE TABLE privacy_options (
    profile_id bigint NOT NULL,
    view_profile character varying(16) DEFAULT 'everyone'::character varying NOT NULL,
    CONSTRAINT privacy_options_view_profile_check CHECK (((view_profile)::text = ANY (ARRAY[('everyone'::character varying)::text, ('members'::character varying)::text, ('followed_by'::character varying)::text, ('none'::character varying)::text])))
);

ALTER TABLE development.privacy_options OWNER TO microcosm;

COMMENT ON TABLE privacy_options IS 'All privacy options for a profile.

At the moment this is just a global switch as to whether the profile as a whole can be viewed and if so by whom. But could easily be extended to include per-field privacy for parts of the profile information and activity.';

CREATE TABLE profiles (
    profile_id bigint NOT NULL,
    site_id bigint NOT NULL,
    user_id bigint NOT NULL,
    profile_name character varying(50) NOT NULL,
    gender character varying(16),
    is_visible boolean DEFAULT true NOT NULL,
    style_id integer,
    item_count integer DEFAULT 0 NOT NULL,
    comment_count integer DEFAULT 0 NOT NULL,
    created timestamp without time zone NOT NULL,
    last_active timestamp without time zone NOT NULL,
    avatar_id bigint,
    yay_count integer DEFAULT 0 NOT NULL,
    meh_count integer DEFAULT 0 NOT NULL,
    grr_count integer DEFAULT 0 NOT NULL,
    avatar_url character varying(125),
    unread_huddles integer DEFAULT 0 NOT NULL
);

ALTER TABLE development.profiles OWNER TO microcosm;

COMMENT ON TABLE profiles IS 'A given user can have any number of profiles, not just across sites (they can have different ways of presenting themselves on different communities) but also within a site (the best way to manage aliases is to permit them but for the admins to know who they really are for the sake of spam and troll fighting).

NOTE: 2013-03-27: Right now the unique constraint on site_id + user_id prevents multiple profiles per site.';

CREATE TABLE users (
    user_id bigint NOT NULL,
    email character varying(254) NOT NULL,
    gender character varying(16),
    language character varying(10) NOT NULL,
    created timestamp without time zone NOT NULL,
    state character varying(20),
    is_banned boolean DEFAULT false NOT NULL,
    password character varying(128) NOT NULL,
    password_date timestamp without time zone NOT NULL,
    dob_day integer,
    dob_month integer,
    dob_year integer,
    CONSTRAINT users_state_check CHECK (((state)::text = ANY (ARRAY[('email_confirm'::character varying)::text, ('valid'::character varying)::text, ('email_confirm_edit'::character varying)::text])))
);

ALTER TABLE development.users OWNER TO microcosm;

COMMENT ON TABLE users IS 'Underlying accounts associated to an email address (the login).

One account can have many profiles (both across sites, and within a site).';

CREATE VIEW profile_filter AS
    SELECT p.site_id, p.profile_id, p.profile_name, u.email, p.gender, p.item_count, p.comment_count, p.created FROM (profiles p JOIN users u ON ((u.user_id = p.user_id)));

ALTER TABLE development.profile_filter OWNER TO microcosm;

CREATE TABLE profile_options (
    profile_id bigint NOT NULL,
    show_dob_year boolean DEFAULT true NOT NULL,
    show_dob_date boolean DEFAULT true NOT NULL,
    is_discouraged boolean DEFAULT false NOT NULL,
    send_email boolean,
    send_sms boolean
);

ALTER TABLE development.profile_options OWNER TO microcosm;

COMMENT ON TABLE profile_options IS 'Profile options, such as whether or not to be emailed, what data to show... this is mostly a set of switches for turning preferences and off throughout the site.';

CREATE SEQUENCE profiles_profile_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.profiles_profile_id_seq OWNER TO microcosm;

ALTER SEQUENCE profiles_profile_id_seq OWNED BY profiles.profile_id;

CREATE TABLE read (
    read_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    read timestamp without time zone NOT NULL
);

ALTER TABLE development.read OWNER TO microcosm;

COMMENT ON TABLE read IS 'In one table, how to keep track of all read, and unread content on a site.

Some things trump others. A read microcosm trumps older unread conversations. So querying is important to get right.';

COMMENT ON COLUMN read.item_id IS 'item_id is *NOT* to be foreign keyed to any particular table.

Depending on item_type_id this column contains a value relating to one of the tables implied by that.

That is to say, if item_type_id states that the item type is a comment then the item_id is a value that is a valid comments.comment_id.

And if the item_type_id states that the item type is a conversation then the item_id is a value that is a valid conversations.conversation_id.';

CREATE SEQUENCE read_read_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.read_read_id_seq OWNER TO microcosm;

ALTER SEQUENCE read_read_id_seq OWNED BY read.read_id;

CREATE TABLE revision_links (
    revlinks_id bigint NOT NULL,
    revision_id bigint NOT NULL,
    link_id bigint NOT NULL
);

ALTER TABLE development.revision_links OWNER TO microcosm;

COMMENT ON TABLE revision_links IS 'Which hyperlinks appear in which revisions?';

CREATE SEQUENCE revision_links_revlinks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.revision_links_revlinks_id_seq OWNER TO microcosm;

ALTER SEQUENCE revision_links_revlinks_id_seq OWNED BY revision_links.revlinks_id;

CREATE TABLE revisions (
    revision_id bigint NOT NULL,
    comment_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    raw text NOT NULL,
    html text,
    created timestamp without time zone NOT NULL,
    is_current boolean DEFAULT true NOT NULL
);

ALTER TABLE development.revisions OWNER TO microcosm;

COMMENT ON TABLE revisions IS 'Version controlled comments. Effectively every comment will have an edit history, this is part of troll fighting as comments should be editable, but that allows people to be offensive and inflammatory and then cover their tracks by editing their post. There is some debate as to whether the history should be public, given that if an edit is requested to remove copyrighted content then we shouldn''t continue to serve that content. It may be that we restrict viewing is_current = FALSE items to moderators and admins only.

raw will contain the markdown formatted comment.

html is a nukeable column and will contain the HTML rendered content to prevent re-parsing the markdown.';

CREATE SEQUENCE revisions_revision_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.revisions_revision_id_seq OWNER TO microcosm;

ALTER SEQUENCE revisions_revision_id_seq OWNED BY revisions.revision_id;

CREATE TABLE rewrite_domain_rules (
    domain_id bigint NOT NULL,
    rule_id bigint NOT NULL
);

ALTER TABLE development.rewrite_domain_rules OWNER TO microcosm;

CREATE TABLE rewrite_domains (
    domain_id bigint NOT NULL,
    domain_regex character varying(512) NOT NULL
);

ALTER TABLE development.rewrite_domains OWNER TO microcosm;

CREATE SEQUENCE rewrite_domains_domain_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.rewrite_domains_domain_id_seq OWNER TO microcosm;

ALTER SEQUENCE rewrite_domains_domain_id_seq OWNED BY rewrite_domains.domain_id;

CREATE TABLE rewrite_rules (
    rule_id bigint NOT NULL,
    name character varying(64) NOT NULL,
    match_regex text NOT NULL,
    replace_regex text NOT NULL,
    is_enabled boolean DEFAULT false,
    sequence integer DEFAULT 99 NOT NULL
);

ALTER TABLE development.rewrite_rules OWNER TO microcosm;

COMMENT ON TABLE rewrite_rules IS 'This is a table that contains regular expressions to match and replace content embedded in the comments.

Effectively, this table contains the rules that will embed stuff (like YouTube) without requiring that we visit the site.';

CREATE TABLE role_members_cache (
    site_id bigint NOT NULL,
    microcosm_id bigint,
    role_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    in_role boolean DEFAULT true NOT NULL
);

ALTER TABLE development.role_members_cache OWNER TO microcosm;

CREATE TABLE role_profiles (
    role_id bigint NOT NULL,
    profile_id bigint NOT NULL
);

ALTER TABLE development.role_profiles OWNER TO microcosm;

COMMENT ON TABLE role_profiles IS 'Maps a profile to a role such that the profile becomes a member of the role.';

CREATE TABLE roles (
    role_id bigint NOT NULL,
    title character varying(50) NOT NULL,
    site_id bigint NOT NULL,
    microcosm_id bigint,
    created timestamp without time zone NOT NULL,
    created_by bigint,
    edited timestamp without time zone,
    edited_by bigint,
    edit_reason character varying(150),
    is_moderator_role boolean DEFAULT false NOT NULL,
    is_banned_role boolean DEFAULT false NOT NULL,
    include_guests boolean DEFAULT false NOT NULL,
    can_read boolean NOT NULL,
    can_create boolean NOT NULL,
    can_update boolean NOT NULL,
    can_delete boolean NOT NULL,
    can_close_own boolean NOT NULL,
    can_open_own boolean NOT NULL,
    can_read_others boolean NOT NULL,
    include_users boolean DEFAULT false NOT NULL,
    CONSTRAINT roles_check CHECK ((((is_moderator_role IS TRUE) AND (is_banned_role IS TRUE)) IS FALSE))
);

ALTER TABLE development.roles OWNER TO microcosm;

COMMENT ON TABLE roles IS 'Describes a set of permissions related to Microcosms, or a default set of permissions used by Microcosms on a site';

COMMENT ON COLUMN roles.title IS 'Arbritrary title given to the role to help the admin and moderators refer to the role later';

COMMENT ON COLUMN roles.site_id IS 'Roles are site specific, this defines which site.';

COMMENT ON COLUMN roles.microcosm_id IS 'If NULL then this is one of the default roles for the site

If NOT NULL then this relates to a Microcosm and describes a role for that Microcosm';

COMMENT ON COLUMN roles.created_by IS 'This is NOT NULL, but create_owned_site requires it to be NULL for the time it takes to solve the chicken and egg problem of creating both a profile and site whilst none exist.';

COMMENT ON COLUMN roles.is_moderator_role IS 'If true, then anyone assigned this role has access to the moderation queue as well as access to the Microcosm admin control panel (role and permission management)';

COMMENT ON COLUMN roles.is_banned_role IS 'If true, then people assigned this role cannot access anything in the Microcosm';

COMMENT ON COLUMN roles.can_read IS 'Can this user read items within the Microcosm?

If false, then all other permissions are implicitly false as none could be performed without a general read flag, and this is tantamount to making the Microcosm hidden or private.';

COMMENT ON COLUMN roles.can_create IS 'Can this user create items within the Microcosm?';

COMMENT ON COLUMN roles.can_update IS 'Can this user update their own items within a Microcosm?';

COMMENT ON COLUMN roles.can_delete IS 'Can this user delete their own items within a Microcosm?';

COMMENT ON COLUMN roles.can_close_own IS 'Can this user close items they own within a Microcosm?';

COMMENT ON COLUMN roles.can_open_own IS 'Can this user open items they own in a Microcosm?';

COMMENT ON COLUMN roles.can_read_others IS 'Can this user read items owned/started by non-moderaters, within a Microcosm?';

CREATE SEQUENCE roles_role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.roles_role_id_seq OWNER TO microcosm;

ALTER SEQUENCE roles_role_id_seq OWNED BY roles.role_id;

CREATE TABLE search_index (
    site_id bigint NOT NULL,
    microcosm_id bigint,
    profile_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    parent_item_type_id bigint,
    parent_item_id bigint,
    title_text text,
    title_vector tsvector,
    document_text text NOT NULL,
    document_vector tsvector NOT NULL,
    last_modified timestamp without time zone DEFAULT now() NOT NULL
);

ALTER TABLE development.search_index OWNER TO microcosm;

CREATE TABLE site_options (
    site_id bigint NOT NULL,
    send_email boolean,
    send_sms boolean,
    only_admins_can_create_microcosms boolean DEFAULT false NOT NULL
);

ALTER TABLE development.site_options OWNER TO microcosm;

CREATE TABLE site_stats (
    site_id bigint NOT NULL,
    active_profiles bigint NOT NULL,
    online_profiles bigint NOT NULL,
    total_profiles bigint NOT NULL,
    total_conversations bigint NOT NULL,
    total_events bigint NOT NULL,
    total_comments bigint NOT NULL
);

ALTER TABLE development.site_stats OWNER TO microcosm;

CREATE TABLE sites (
    site_id bigint NOT NULL,
    title character varying(50) NOT NULL,
    description text,
    subdomain_key character varying(50) NOT NULL,
    domain character varying(253),
    created timestamp without time zone NOT NULL,
    created_by bigint NOT NULL,
    owned_by bigint NOT NULL,
    theme_id bigint NOT NULL,
    logo_url character varying(2000),
    background_url character varying(2000),
    ga_web_property_id character varying(15),
    background_color character varying(50) DEFAULT '#FFFFFF'::character varying NOT NULL,
    background_position character varying(6) DEFAULT 'cover'::character varying NOT NULL,
    link_color character varying(50) DEFAULT '#4082C3'::character varying NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    favicon_url character varying(2000)
);

ALTER TABLE development.sites OWNER TO microcosm;

COMMENT ON TABLE sites IS 'Basic knowledge of which sites exist. These are essentially collections of microcosms for a given URL with an assigned administrator. Dull stuff.';

COMMENT ON COLUMN sites.domain IS 'If NULL then all requests to this site must be in the form https?://<subdomain_key>.microcosm.app/

If NOT NULL then all requests to this site must be in the form http://<domain>/';

CREATE SEQUENCE sites_site_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.sites_site_id_seq OWNER TO microcosm;

ALTER SEQUENCE sites_site_id_seq OWNED BY sites.site_id;

CREATE TABLE themes (
    theme_id bigint NOT NULL,
    title character varying(256) NOT NULL,
    logo_url character varying(2000) NOT NULL,
    background_url character varying(2000) NOT NULL,
    background_color character varying(50) DEFAULT '#FFFFFF'::character varying NOT NULL,
    background_position character varying(6) DEFAULT 'cover'::character varying NOT NULL,
    link_color character varying(50) DEFAULT '#4082C3'::character varying NOT NULL,
    favicon_url character varying(2000) DEFAULT '/static/img/favico.png'::character varying NOT NULL
);

ALTER TABLE development.themes OWNER TO microcosm;

COMMENT ON TABLE themes IS 'A theme is paired to a microweb collection of static files for Bootstrap CSS, as well as having override values allowing the logo and header to be changed per site';

CREATE SEQUENCE themes_theme_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.themes_theme_id_seq OWNER TO microcosm;

ALTER SEQUENCE themes_theme_id_seq OWNED BY themes.theme_id;

CREATE TABLE update_options (
    profile_id bigint NOT NULL,
    update_type_id bigint NOT NULL,
    send_email boolean DEFAULT false NOT NULL,
    send_sms boolean DEFAULT false NOT NULL
);

ALTER TABLE development.update_options OWNER TO microcosm;

COMMENT ON TABLE update_options IS 'The system will react to all changes and want to send updates to those people who are observers of an event.

This table keeps a track of which profiles have opted in and out of receiving the updates, such as "someone replied to a comment you made".';

CREATE TABLE update_options_defaults (
    update_type_id bigint NOT NULL,
    send_email boolean DEFAULT false NOT NULL,
    send_sms boolean DEFAULT false NOT NULL
);

ALTER TABLE development.update_options_defaults OWNER TO microcosm;

COMMENT ON TABLE update_options_defaults IS 'Encapsulates the default options for updates for a given update type. Applies if no overriding preference exists in the update_options table for the current profile';

CREATE TABLE update_types (
    update_type_id bigint NOT NULL,
    title character varying(50) NOT NULL,
    description character varying(512) NOT NULL,
    email_subject character varying(255) NOT NULL,
    email_body_text text NOT NULL,
    email_body_html text NOT NULL
);

ALTER TABLE development.update_types OWNER TO microcosm;

COMMENT ON TABLE update_types IS 'This table keeps track of the kind of updates that can be sent, for example: "someone replied to a comment", "someone ymg a comment", "new people enrolled to the event you''re watching", etc';

CREATE TABLE updates_latest (
    update_id bigint NOT NULL
);

ALTER TABLE development.updates_latest OWNER TO microcosm;

CREATE SEQUENCE url_rewrites_url_rewrite_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.url_rewrites_url_rewrite_id_seq OWNER TO microcosm;

ALTER SEQUENCE url_rewrites_url_rewrite_id_seq OWNED BY rewrite_rules.rule_id;

CREATE SEQUENCE users_user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.users_user_id_seq OWNER TO microcosm;

ALTER SEQUENCE users_user_id_seq OWNED BY users.user_id;

CREATE TABLE value_types (
    value_type_id bigint NOT NULL,
    title character varying(50) NOT NULL
);

ALTER TABLE development.value_types OWNER TO microcosm;

CREATE TABLE views (
    item_type_id bigint DEFAULT 0 NOT NULL,
    item_id bigint DEFAULT 0 NOT NULL
);

ALTER TABLE development.views OWNER TO microcosm;

COMMENT ON TABLE views IS 'Do not FK this table to anything, we need INSERTs to be fast and non-checking and non-blocking.

Purpose of this table is to allow us to count views and async update the item table as well as events|conversations|polls|etc.

Basically every row is one view, counting identical tuples reveals the sum of views against the given item type and item, and then value can then be stuffed into the items table, and reverse copied out to the other tables.';

CREATE TABLE votes (
    user_id bigint NOT NULL,
    choice_id bigint NOT NULL,
    poll_id bigint NOT NULL,
    voted timestamp without time zone NOT NULL
);

ALTER TABLE development.votes OWNER TO microcosm;

COMMENT ON TABLE votes IS 'Votes on a poll.

These are just the actual registered user votes, anon users would increment the counts on the polls table without creating an entry in here.

Interesting thing: This is one of the very few tables that joins to user_id rather than profile_id. The reason for this is to prevent people gaming polls. A given user can only vote once regardless of the number of profiles that they may have on a site.';

CREATE TABLE watchers (
    watcher_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    last_notified timestamp without time zone,
    send_email boolean DEFAULT false NOT NULL,
    send_sms boolean DEFAULT false NOT NULL
);

ALTER TABLE development.watchers OWNER TO microcosm;

CREATE SEQUENCE watchers_watcher_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.watchers_watcher_id_seq OWNER TO microcosm;

ALTER SEQUENCE watchers_watcher_id_seq OWNED BY watchers.watcher_id;

CREATE TABLE ymg (
    ymg_id bigint NOT NULL,
    item_type_id bigint NOT NULL,
    item_id bigint NOT NULL,
    profile_id bigint NOT NULL,
    created timestamp without time zone NOT NULL,
    item_profile_id bigint NOT NULL,
    value integer NOT NULL
);

ALTER TABLE development.ymg OWNER TO microcosm;

COMMENT ON TABLE ymg IS 'Yay Meh Grr

value = +1 = Yay
value = 0 = Meh
value = -1 = Grr

item_profile_id is on here too, to simplify notifications of the kind: "Someone +1''d your thing"';

COMMENT ON COLUMN ymg.item_id IS 'item_id is *NOT* to be foreign keyed to any particular table.

Depending on item_type_id this column contains a value relating to one of the tables implied by that.

That is to say, if item_type_id states that the item type is a comment then the item_id is a value that is a valid comments.comment_id.

And if the item_type_id states that the item type is a conversation then the item_id is a value that is a valid conversations.conversation_id.';

CREATE SEQUENCE ymg_ymg_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE development.ymg_ymg_id_seq OWNER TO microcosm;

ALTER SEQUENCE ymg_ymg_id_seq OWNED BY ymg.ymg_id;

ALTER TABLE ONLY access_tokens ALTER COLUMN access_token_id SET DEFAULT nextval('access_tokens_access_token_id_seq'::regclass);

ALTER TABLE ONLY attachment_meta ALTER COLUMN attachment_meta_id SET DEFAULT nextval('attachment_meta_attachment_meta_id_seq'::regclass);

ALTER TABLE ONLY attachments ALTER COLUMN attachment_id SET DEFAULT nextval('attachments_attachment_id_seq'::regclass);

ALTER TABLE ONLY attendees ALTER COLUMN attendee_id SET DEFAULT nextval('attendees_attendee_id_seq'::regclass);

ALTER TABLE ONLY attribute_keys ALTER COLUMN attribute_id SET DEFAULT nextval('attributes_attribute_id_seq'::regclass);

ALTER TABLE ONLY bans ALTER COLUMN ban_id SET DEFAULT nextval('bans_ban_id_seq'::regclass);

ALTER TABLE ONLY choices ALTER COLUMN choice_id SET DEFAULT nextval('choices_choice_id_seq'::regclass);

ALTER TABLE ONLY comments ALTER COLUMN comment_id SET DEFAULT nextval('comments_comment_id_seq'::regclass);

ALTER TABLE ONLY conversations ALTER COLUMN conversation_id SET DEFAULT nextval('conversations_conversation_id_seq'::regclass);

ALTER TABLE ONLY criteria ALTER COLUMN criteria_id SET DEFAULT nextval('criteria_criteria_id_seq'::regclass);

ALTER TABLE ONLY events ALTER COLUMN event_id SET DEFAULT nextval('events_event_id_seq'::regclass);

ALTER TABLE ONLY huddles ALTER COLUMN huddle_id SET DEFAULT nextval('huddles_huddle_id_seq'::regclass);

ALTER TABLE ONLY import_origins ALTER COLUMN origin_id SET DEFAULT nextval('import_origins_origin_id_seq'::regclass);

ALTER TABLE ONLY links ALTER COLUMN link_id SET DEFAULT nextval('links_link_id_seq'::regclass);

ALTER TABLE ONLY menus ALTER COLUMN menu_id SET DEFAULT nextval('menus_menu_id_seq'::regclass);

ALTER TABLE ONLY microcosm_profile_options ALTER COLUMN microcosm_profile_option_id SET DEFAULT nextval('microcosm_profile_options_microcosm_profile_option_id_seq'::regclass);

ALTER TABLE ONLY microcosms ALTER COLUMN microcosm_id SET DEFAULT nextval('microcosms_microcosm_id_seq'::regclass);

ALTER TABLE ONLY moderation_queue ALTER COLUMN moderation_queue_id SET DEFAULT nextval('moderation_queue_moderation_queue_id_seq'::regclass);

ALTER TABLE ONLY oauth_clients ALTER COLUMN client_id SET DEFAULT nextval('oauth_clients_client_id_seq'::regclass);

ALTER TABLE ONLY polls ALTER COLUMN poll_id SET DEFAULT nextval('polls_poll_id_seq'::regclass);

ALTER TABLE ONLY profiles ALTER COLUMN profile_id SET DEFAULT nextval('profiles_profile_id_seq'::regclass);

ALTER TABLE ONLY read ALTER COLUMN read_id SET DEFAULT nextval('read_read_id_seq'::regclass);

ALTER TABLE ONLY revision_links ALTER COLUMN revlinks_id SET DEFAULT nextval('revision_links_revlinks_id_seq'::regclass);

ALTER TABLE ONLY revisions ALTER COLUMN revision_id SET DEFAULT nextval('revisions_revision_id_seq'::regclass);

ALTER TABLE ONLY rewrite_domains ALTER COLUMN domain_id SET DEFAULT nextval('rewrite_domains_domain_id_seq'::regclass);

ALTER TABLE ONLY rewrite_rules ALTER COLUMN rule_id SET DEFAULT nextval('url_rewrites_url_rewrite_id_seq'::regclass);

ALTER TABLE ONLY roles ALTER COLUMN role_id SET DEFAULT nextval('roles_role_id_seq'::regclass);

ALTER TABLE ONLY sites ALTER COLUMN site_id SET DEFAULT nextval('sites_site_id_seq'::regclass);

ALTER TABLE ONLY themes ALTER COLUMN theme_id SET DEFAULT nextval('themes_theme_id_seq'::regclass);

ALTER TABLE ONLY updates ALTER COLUMN update_id SET DEFAULT nextval('alerts_alert_id_seq'::regclass);

ALTER TABLE ONLY users ALTER COLUMN user_id SET DEFAULT nextval('users_user_id_seq'::regclass);

ALTER TABLE ONLY watchers ALTER COLUMN watcher_id SET DEFAULT nextval('watchers_watcher_id_seq'::regclass);

ALTER TABLE ONLY ymg ALTER COLUMN ymg_id SET DEFAULT nextval('ymg_ymg_id_seq'::regclass);

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_pkey PRIMARY KEY (access_token_id);

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_token_value_key UNIQUE (token_value);

ALTER TABLE ONLY admins
    ADD CONSTRAINT admins_pkey PRIMARY KEY (admin_id);

ALTER TABLE ONLY admins
    ADD CONSTRAINT admins_site_id_key UNIQUE (site_id, profile_id);

ALTER TABLE ONLY update_options_defaults
    ADD CONSTRAINT alert_defaults_pkey PRIMARY KEY (update_type_id);

ALTER TABLE ONLY update_options
    ADD CONSTRAINT alert_preferences_pkey PRIMARY KEY (profile_id, update_type_id);

ALTER TABLE ONLY update_types
    ADD CONSTRAINT alert_types_pkey PRIMARY KEY (update_type_id);

ALTER TABLE ONLY updates
    ADD CONSTRAINT alerts_pkey PRIMARY KEY (update_id);

ALTER TABLE ONLY attachments
    ADD CONSTRAINT attachment_id_pk PRIMARY KEY (attachment_id);

ALTER TABLE ONLY attachment_meta
    ADD CONSTRAINT attachment_meta_id_pk PRIMARY KEY (attachment_meta_id);

ALTER TABLE ONLY attachment_views
    ADD CONSTRAINT attachment_view_id_pk PRIMARY KEY (attachment_id);

ALTER TABLE ONLY attendee_state
    ADD CONSTRAINT attendee_state_pkey PRIMARY KEY (state_id);

ALTER TABLE ONLY attendee_state
    ADD CONSTRAINT attendee_state_title_key UNIQUE (title);

ALTER TABLE ONLY attendees
    ADD CONSTRAINT attendees_event_id_profile_id_key UNIQUE (event_id, profile_id);

ALTER TABLE ONLY attendees
    ADD CONSTRAINT attendees_pkey PRIMARY KEY (attendee_id);

ALTER TABLE ONLY attribute_keys
    ADD CONSTRAINT attributes_item_type_id_item_id_name_key UNIQUE (item_type_id, item_id, key);

ALTER TABLE ONLY attribute_keys
    ADD CONSTRAINT attributes_pkey PRIMARY KEY (attribute_id);

ALTER TABLE ONLY banned_emails
    ADD CONSTRAINT banned_emails_pk PRIMARY KEY (site_id, email);

ALTER TABLE ONLY bans
    ADD CONSTRAINT bans_pkey PRIMARY KEY (ban_id);

ALTER TABLE ONLY comments
    ADD CONSTRAINT comments_pkey PRIMARY KEY (comment_id);

ALTER TABLE ONLY conversations
    ADD CONSTRAINT conversations_pkey PRIMARY KEY (conversation_id);

ALTER TABLE ONLY criteria
    ADD CONSTRAINT criteria_pkey PRIMARY KEY (criteria_id);

ALTER TABLE ONLY disabled_roles
    ADD CONSTRAINT disabled_roles_pkey PRIMARY KEY (microcosm_id, role_id);

ALTER TABLE ONLY events
    ADD CONSTRAINT events_pkey PRIMARY KEY (event_id);

ALTER TABLE ONLY flags
    ADD CONSTRAINT flags_pkey PRIMARY KEY (item_type_id, item_id);

ALTER TABLE ONLY follows
    ADD CONSTRAINT follows_pkey PRIMARY KEY (profile_id, follow_profile_id);

ALTER TABLE ONLY huddle_profiles
    ADD CONSTRAINT huddle_recipients_pkey PRIMARY KEY (huddle_id, profile_id);

ALTER TABLE ONLY huddles
    ADD CONSTRAINT huddles_pkey PRIMARY KEY (huddle_id);

ALTER TABLE ONLY ignores
    ADD CONSTRAINT ignores_pkey PRIMARY KEY (profile_id, item_type_id, item_id);

ALTER TABLE ONLY import_origins
    ADD CONSTRAINT import_origins_pkey PRIMARY KEY (origin_id);

ALTER TABLE ONLY import_origins
    ADD CONSTRAINT import_origins_title_key UNIQUE (title);

ALTER TABLE ONLY imported_items
    ADD CONSTRAINT imported_items_pkey PRIMARY KEY (origin_id, item_type_id, old_id);

ALTER TABLE ONLY ips
    ADD CONSTRAINT ips_pkey PRIMARY KEY (site_id, item_type_id, item_id, profile_id, seen);

ALTER TABLE ONLY item_types
    ADD CONSTRAINT item_type_id_pk PRIMARY KEY (item_type_id);

ALTER TABLE ONLY links
    ADD CONSTRAINT links_pkey PRIMARY KEY (link_id);

ALTER TABLE links CLUSTER ON links_pkey;

ALTER TABLE ONLY links
    ADD CONSTRAINT links_short_url_key UNIQUE (short_url);

ALTER TABLE ONLY links
    ADD CONSTRAINT links_url_key UNIQUE (url);

ALTER TABLE ONLY menus
    ADD CONSTRAINT menus_pkey PRIMARY KEY (menu_id);

ALTER TABLE ONLY menus
    ADD CONSTRAINT menus_site_id_sequence_key UNIQUE (site_id, sequence);

ALTER TABLE ONLY metrics
    ADD CONSTRAINT metrics_pkey PRIMARY KEY (job_timestamp);

ALTER TABLE ONLY microcosms
    ADD CONSTRAINT microcosm_id_pk PRIMARY KEY (microcosm_id);

ALTER TABLE ONLY microcosm_options
    ADD CONSTRAINT microcosm_options_pkey PRIMARY KEY (microcosm_id);

ALTER TABLE ONLY microcosm_profile_options
    ADD CONSTRAINT microcosm_profile_options_pkey PRIMARY KEY (microcosm_profile_option_id);

ALTER TABLE ONLY moderation_queue
    ADD CONSTRAINT moderation_queue_pkey PRIMARY KEY (moderation_queue_id);

ALTER TABLE ONLY oauth_clients
    ADD CONSTRAINT oauth_clients_pkey PRIMARY KEY (client_id);

ALTER TABLE ONLY permissions_cache
    ADD CONSTRAINT permissions_cache_pkey PRIMARY KEY (site_id, profile_id, item_type_id, item_id);

ALTER TABLE ONLY choices
    ADD CONSTRAINT poll_choices_pkey PRIMARY KEY (choice_id);

ALTER TABLE ONLY polls
    ADD CONSTRAINT polls_pkey PRIMARY KEY (poll_id);

ALTER TABLE ONLY privacy_options
    ADD CONSTRAINT privacy_options_pkey PRIMARY KEY (profile_id);

ALTER TABLE ONLY profile_options
    ADD CONSTRAINT profile_options_pkey PRIMARY KEY (profile_id);

ALTER TABLE ONLY profiles
    ADD CONSTRAINT profiles_pkey PRIMARY KEY (profile_id);

ALTER TABLE ONLY profiles
    ADD CONSTRAINT profiles_site_id_user_id_key UNIQUE (site_id, user_id);

ALTER TABLE ONLY read
    ADD CONSTRAINT read_id_pk PRIMARY KEY (read_id);

ALTER TABLE ONLY revision_links
    ADD CONSTRAINT revision_links_pkey PRIMARY KEY (revlinks_id);

ALTER TABLE ONLY revisions
    ADD CONSTRAINT revisions_comment_id_created_key UNIQUE (comment_id, created);

ALTER TABLE ONLY revisions
    ADD CONSTRAINT revisions_pkey PRIMARY KEY (revision_id);

ALTER TABLE ONLY rewrite_domain_rules
    ADD CONSTRAINT rewrite_domain_rules_pkey PRIMARY KEY (domain_id, rule_id);

ALTER TABLE ONLY rewrite_domains
    ADD CONSTRAINT rewrite_domains_pkey PRIMARY KEY (domain_id);

ALTER TABLE ONLY role_members_cache
    ADD CONSTRAINT role_members_cache_pkey PRIMARY KEY (site_id, role_id, profile_id);

ALTER TABLE ONLY role_profiles
    ADD CONSTRAINT role_profiles_pkey PRIMARY KEY (role_id, profile_id);

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (role_id);

ALTER TABLE ONLY search_index
    ADD CONSTRAINT search_index_pkey PRIMARY KEY (item_type_id, item_id);

ALTER TABLE ONLY sites
    ADD CONSTRAINT site_id_pk PRIMARY KEY (site_id);

ALTER TABLE ONLY site_stats
    ADD CONSTRAINT site_id_pkey PRIMARY KEY (site_id);

ALTER TABLE ONLY site_options
    ADD CONSTRAINT site_options_pkey PRIMARY KEY (site_id);

ALTER TABLE ONLY sites
    ADD CONSTRAINT sites_domain_key UNIQUE (domain);

ALTER TABLE ONLY sites
    ADD CONSTRAINT sites_subdomain_key_key UNIQUE (subdomain_key);

ALTER TABLE ONLY themes
    ADD CONSTRAINT themes_pkey PRIMARY KEY (theme_id);

ALTER TABLE ONLY updates_latest
    ADD CONSTRAINT updates_latest_pkey PRIMARY KEY (update_id);

ALTER TABLE ONLY rewrite_rules
    ADD CONSTRAINT url_rewrite_id_pk PRIMARY KEY (rule_id);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);

ALTER TABLE ONLY value_types
    ADD CONSTRAINT value_types_pkey PRIMARY KEY (value_type_id);

ALTER TABLE ONLY value_types
    ADD CONSTRAINT value_types_title_key UNIQUE (title);

ALTER TABLE ONLY attribute_values
    ADD CONSTRAINT values_pkey PRIMARY KEY (attribute_id);

ALTER TABLE ONLY votes
    ADD CONSTRAINT votes_pkey PRIMARY KEY (user_id, choice_id);

ALTER TABLE ONLY watchers
    ADD CONSTRAINT watchers_pkey PRIMARY KEY (watcher_id);

ALTER TABLE ONLY watchers
    ADD CONSTRAINT watchers_profile_id_item_type_id_item_id_key UNIQUE (profile_id, item_type_id, item_id);

ALTER TABLE ONLY ymg
    ADD CONSTRAINT ymg_pkey PRIMARY KEY (ymg_id);

CREATE INDEX attachments_itemtypeandid_idx ON attachments USING btree (item_type_id, item_id);

CREATE INDEX attribute_keys_itemtypeandid_idx ON attribute_keys USING btree (item_type_id, item_id);

CREATE INDEX bans_site_id_user_id_idx ON bans USING btree (site_id, user_id);

CREATE INDEX comments_in_reply_to ON comments USING btree (in_reply_to);

CREATE INDEX comments_isdeleted_idx ON comments USING btree (is_deleted);

CREATE INDEX comments_ismoderated_idx ON comments USING btree (is_moderated);

CREATE INDEX comments_itemtypeandid_idx ON comments USING btree (item_type_id, item_id);

CREATE INDEX comments_partial_idx ON comments USING btree (item_type_id, item_id) WHERE (is_deleted IS FALSE);

CREATE INDEX conversations_isdeleted_idx ON conversations USING btree (is_deleted);

CREATE INDEX conversations_microcosmid_idx ON conversations USING btree (microcosm_id);

CREATE INDEX events_isdeleted_idx ON events USING btree (is_deleted);

CREATE INDEX events_microcosmid_idx ON events USING btree (microcosm_id);

CREATE INDEX flags_createdby_idx ON flags USING btree (created_by);

CREATE INDEX flags_deleted_idx ON flags USING btree (site_id, microcosm_is_deleted, microcosm_is_moderated, parent_is_deleted, parent_is_moderated, item_is_deleted, item_is_moderated) WHERE ((((((microcosm_is_deleted IS NOT TRUE) AND (microcosm_is_moderated IS NOT TRUE)) AND (parent_is_deleted IS NOT TRUE)) AND (parent_is_moderated IS NOT TRUE)) AND (item_is_deleted IS NOT TRUE)) AND (item_is_moderated IS NOT TRUE));

CREATE INDEX flags_lastmodified2_idx ON flags USING btree (last_modified) WHERE (item_type_id = ANY (ARRAY[(3)::bigint, (5)::bigint, (6)::bigint, (7)::bigint, (9)::bigint]));

CREATE INDEX flags_lastmodified_idx ON flags USING btree (microcosm_id, last_modified DESC);

CREATE INDEX flags_microcosmid_idx ON flags USING btree (microcosm_id);

CREATE INDEX flags_parentitemtypeidandparentitemid_idx ON flags USING btree (parent_item_type_id, parent_item_id);

CREATE INDEX flags_partial_idx ON flags USING btree (item_type_id, item_id) WHERE ((NOT item_is_deleted) AND (NOT item_is_moderated));

CREATE INDEX flags_partial_item_idx ON flags USING btree (microcosm_id, item_type_id, item_id, last_modified DESC);

CREATE INDEX flags_site_id_sitemap_index_idx ON flags USING btree (site_id, sitemap_index DESC NULLS LAST);

CREATE INDEX huddles_huddle_id_created_by_idx ON huddles USING btree (huddle_id, created_by);

CREATE INDEX huddles_huddleprofileid_idx ON huddle_profiles USING btree (huddle_id, profile_id);

CREATE INDEX huddles_profileid_idx ON huddle_profiles USING btree (profile_id);

CREATE INDEX ignores_item_type_id_item_id_idx ON ignores USING btree (item_type_id, item_id);

CREATE INDEX ignores_profile_id_idx ON ignores USING btree (profile_id);

CREATE INDEX imported_items_idx ON imported_items USING btree (origin_id, item_type_id, ((old_id)::bigint));

CREATE INDEX ips_action_idx ON ips USING btree (action);

CREATE INDEX ips_ip_idx ON ips USING btree (ip);

CREATE INDEX ips_profileid_idx ON ips USING btree (profile_id);

CREATE INDEX ips_seen_idx ON ips USING btree (item_type_id, item_id);

CREATE INDEX ips_siteid_idx ON ips USING btree (site_id);

CREATE INDEX item_types_rank_idx ON item_types USING btree (item_type_id, rank DESC);

CREATE INDEX links_short_url_idx ON links USING btree (short_url);

CREATE INDEX links_url_idx ON links USING btree (url);

CREATE INDEX microcosms_created_idx ON microcosms USING btree (created);

CREATE INDEX microcosms_is_deleted_idx ON microcosms USING btree (is_deleted);

CREATE INDEX microcosms_isdeleted_idx ON microcosms USING btree (is_deleted);

CREATE INDEX microcosms_last_comment_created_idx ON microcosms USING btree (last_comment_created);

CREATE INDEX microcosms_microcosm_id_created_by_owned_by_idx ON microcosms USING btree (microcosm_id, created_by, owned_by);

CREATE INDEX moderation_queue_itemtypeandid_idx ON moderation_queue USING btree (item_type_id, item_id);

CREATE INDEX parent_modified_idx ON flags USING btree (parent_item_type_id, parent_item_id, last_modified);

CREATE INDEX polls_isdeleted_idx ON polls USING btree (is_deleted);

CREATE INDEX polls_microcosmid_idx ON polls USING btree (microcosm_id);

CREATE INDEX profiles_siteid_idx ON profiles USING btree (site_id);

CREATE INDEX read_partial_idx ON read USING btree (item_type_id, item_id, profile_id);

CREATE INDEX read_partial_order_idx ON read USING btree (item_type_id, item_id, profile_id, read DESC);

CREATE INDEX read_profileitemtypeandid_idx ON read USING btree (profile_id, item_type_id, item_id);

CREATE INDEX read_profileitemtypeid_idx ON read USING btree (profile_id, item_type_id);

CREATE INDEX revisionlinks_linkid_idx ON revision_links USING btree (revision_id, link_id);

CREATE INDEX revisions_comment_idx ON revisions USING btree (comment_id) WHERE (is_current = true);

CREATE INDEX revisions_profile_idx ON revisions USING btree (profile_id);

CREATE INDEX searchindex_documentvector_idx ON search_index USING gin (document_vector);

CREATE INDEX searchindex_itemtypeidanditemid_idx ON search_index USING btree (item_type_id, item_id);

CREATE INDEX searchindex_parentitemtypeidandparentitemid_idx ON search_index USING btree (parent_item_type_id, parent_item_id);

CREATE INDEX searchindex_profileid_idx ON search_index USING btree (profile_id);

CREATE INDEX searchindex_siteid_idx ON search_index USING btree (site_id);

CREATE INDEX searchindex_titlevector_idx ON search_index USING gin (title_vector);

CREATE INDEX sites_site_id_owned_by_created_by_idx ON sites USING btree (site_id, owned_by, created_by);

CREATE INDEX update_options_profile_id_idx ON update_options USING btree (profile_id);

CREATE INDEX updates_forprofile_idx ON updates USING btree (for_profile_id, update_type_id, item_type_id, item_id);

CREATE INDEX updates_itemtypeandid_idx ON updates USING btree (item_type_id, item_id);

CREATE INDEX updates_parentitemtypeandid_idx ON updates USING btree (for_profile_id, parent_item_type_id, parent_item_id) WHERE (update_type_id = ANY (ARRAY[(1)::bigint, (4)::bigint]));

CREATE INDEX updates_updatetypeid_idx ON updates USING btree (update_type_id, created DESC);

CREATE INDEX views_item_idx ON views USING btree (item_id);

CREATE INDEX views_itemtype_idx ON views USING btree (item_type_id);

CREATE INDEX watchers_itemtypeandid_idx ON watchers USING btree (item_type_id, item_id);

CREATE INDEX watchers_profile_id_item_type_id_item_id_idx ON watchers USING btree (profile_id, item_type_id, item_id);

CREATE INDEX watchers_profileanditemtype_idx ON watchers USING btree (profile_id, item_type_id);

CREATE TRIGGER comments_flags AFTER INSERT OR DELETE OR UPDATE ON comments FOR EACH ROW EXECUTE PROCEDURE update_comments_flags();

CREATE TRIGGER comments_search_index AFTER INSERT OR DELETE OR UPDATE ON comments FOR EACH ROW EXECUTE PROCEDURE update_comments_search_index();

CREATE TRIGGER conversations_flags AFTER INSERT OR DELETE OR UPDATE ON conversations FOR EACH ROW EXECUTE PROCEDURE update_conversations_flags();

CREATE TRIGGER conversations_search_index AFTER INSERT OR DELETE OR UPDATE ON conversations FOR EACH ROW EXECUTE PROCEDURE update_conversations_search_index();

CREATE TRIGGER events_flags AFTER INSERT OR DELETE OR UPDATE ON events FOR EACH ROW EXECUTE PROCEDURE update_events_flags();

CREATE TRIGGER events_search_index AFTER INSERT OR DELETE OR UPDATE ON events FOR EACH ROW EXECUTE PROCEDURE update_events_search_index();

CREATE TRIGGER huddles_flags AFTER INSERT OR DELETE OR UPDATE ON huddles FOR EACH ROW EXECUTE PROCEDURE update_huddles_flags();

CREATE TRIGGER huddles_search_index AFTER INSERT OR DELETE OR UPDATE ON huddles FOR EACH ROW EXECUTE PROCEDURE update_huddles_search_index();

CREATE TRIGGER microcosms_flags AFTER INSERT OR DELETE OR UPDATE ON microcosms FOR EACH ROW EXECUTE PROCEDURE update_microcosms_flags();

CREATE TRIGGER microcosms_search_index AFTER INSERT OR DELETE OR UPDATE ON microcosms FOR EACH ROW EXECUTE PROCEDURE update_microcosms_search_index();

CREATE TRIGGER polls_flags AFTER INSERT OR DELETE OR UPDATE ON polls FOR EACH ROW EXECUTE PROCEDURE update_polls_flags();

CREATE TRIGGER profiles_flags AFTER INSERT OR DELETE OR UPDATE ON profiles FOR EACH ROW EXECUTE PROCEDURE update_profiles_flags();

CREATE TRIGGER profiles_search_index AFTER INSERT OR DELETE OR UPDATE ON profiles FOR EACH ROW EXECUTE PROCEDURE update_profiles_search_index();

CREATE TRIGGER revisions_search_index AFTER INSERT OR DELETE OR UPDATE ON revisions FOR EACH ROW EXECUTE PROCEDURE update_revisions_search_index();

CREATE TRIGGER sites_flags AFTER INSERT OR DELETE OR UPDATE ON sites FOR EACH ROW EXECUTE PROCEDURE update_sites_flags();

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_client_id_fkey FOREIGN KEY (client_id) REFERENCES oauth_clients(client_id);

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE ONLY activity_scores
    ADD CONSTRAINT activity_scores_item_type_id_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY activity_scores
    ADD CONSTRAINT activity_scores_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY admins
    ADD CONSTRAINT admins_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY admins
    ADD CONSTRAINT admins_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY update_options_defaults
    ADD CONSTRAINT alert_defaults_alert_type_id_fkey FOREIGN KEY (update_type_id) REFERENCES update_types(update_type_id);

ALTER TABLE ONLY update_options
    ADD CONSTRAINT alert_preferences_alert_type_id_fkey FOREIGN KEY (update_type_id) REFERENCES update_types(update_type_id);

ALTER TABLE ONLY update_options
    ADD CONSTRAINT alert_preferences_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profile_options(profile_id);

ALTER TABLE ONLY updates
    ADD CONSTRAINT alerts_alert_type_id_fkey FOREIGN KEY (update_type_id) REFERENCES update_types(update_type_id);

ALTER TABLE ONLY updates
    ADD CONSTRAINT alerts_alerted_profile_id_fkey FOREIGN KEY (for_profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY updates
    ADD CONSTRAINT alerts_item_type_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY updates
    ADD CONSTRAINT alerts_profile_id_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY updates
    ADD CONSTRAINT alerts_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY attachment_views
    ADD CONSTRAINT attachment_views_attachment_id_fkey FOREIGN KEY (attachment_id) REFERENCES attachments(attachment_id);

ALTER TABLE ONLY attachments
    ADD CONSTRAINT attachments_attachment_meta_id_fkey FOREIGN KEY (attachment_meta_id) REFERENCES attachment_meta(attachment_meta_id);

ALTER TABLE ONLY attachments
    ADD CONSTRAINT attachments_item_type_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY attachments
    ADD CONSTRAINT attachments_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY attendees
    ADD CONSTRAINT attendees_created_by_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY attendees
    ADD CONSTRAINT attendees_edited_by_fkey FOREIGN KEY (edited_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY attendees
    ADD CONSTRAINT attendees_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(event_id);

ALTER TABLE ONLY attendees
    ADD CONSTRAINT attendees_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY attendees
    ADD CONSTRAINT attendees_state_id_fkey FOREIGN KEY (state_id) REFERENCES attendee_state(state_id);

ALTER TABLE ONLY attribute_keys
    ADD CONSTRAINT attributes_item_type_id_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY banned_emails
    ADD CONSTRAINT banned_emails_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY banned_emails
    ADD CONSTRAINT banned_emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE ONLY bans
    ADD CONSTRAINT bans_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY bans
    ADD CONSTRAINT bans_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE ONLY comments
    ADD CONSTRAINT comments_item_type_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY comments
    ADD CONSTRAINT comments_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY conversations
    ADD CONSTRAINT conversations_created_by_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY conversations
    ADD CONSTRAINT conversations_edited_by_fkey FOREIGN KEY (edited_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY conversations
    ADD CONSTRAINT conversations_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY criteria
    ADD CONSTRAINT criteria_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(role_id);

ALTER TABLE ONLY disabled_roles
    ADD CONSTRAINT disabled_roles_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY disabled_roles
    ADD CONSTRAINT disabled_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(role_id);

ALTER TABLE ONLY events
    ADD CONSTRAINT events_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY flags
    ADD CONSTRAINT flags_created_by_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY flags
    ADD CONSTRAINT flags_item_type_id_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY flags
    ADD CONSTRAINT flags_parent_item_type_id_fkey FOREIGN KEY (parent_item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY flags
    ADD CONSTRAINT flags_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY follows
    ADD CONSTRAINT follows_follow_profile_id_fkey FOREIGN KEY (follow_profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY follows
    ADD CONSTRAINT follows_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY huddle_profiles
    ADD CONSTRAINT huddle_recipients_huddle_id_fkey FOREIGN KEY (huddle_id) REFERENCES huddles(huddle_id);

ALTER TABLE ONLY huddle_profiles
    ADD CONSTRAINT huddle_recipients_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY huddles
    ADD CONSTRAINT huddles_created_by_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY huddles
    ADD CONSTRAINT huddles_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY ignores
    ADD CONSTRAINT ignores_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY import_origins
    ADD CONSTRAINT import_origins_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY imported_items
    ADD CONSTRAINT imported_items_item_type_id_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY imported_items
    ADD CONSTRAINT imported_items_origin_id_fkey FOREIGN KEY (origin_id) REFERENCES import_origins(origin_id);

ALTER TABLE ONLY menus
    ADD CONSTRAINT menus_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY microcosm_options
    ADD CONSTRAINT microcosm_options_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY microcosm_profile_options
    ADD CONSTRAINT microcosm_profile_options_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY microcosm_profile_options
    ADD CONSTRAINT microcosm_profile_options_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY microcosms
    ADD CONSTRAINT microcosms_created_by_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY microcosms
    ADD CONSTRAINT microcosms_last_comment_created_by_fkey FOREIGN KEY (last_comment_created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY microcosms
    ADD CONSTRAINT microcosms_last_comment_item_type_id_fkey FOREIGN KEY (last_comment_item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY microcosms
    ADD CONSTRAINT microcosms_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY moderation_queue
    ADD CONSTRAINT moderation_queue_item_type_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY moderation_queue
    ADD CONSTRAINT moderation_queue_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY moderation_queue
    ADD CONSTRAINT moderation_queue_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY choices
    ADD CONSTRAINT poll_choices_poll_id_fkey FOREIGN KEY (poll_id) REFERENCES polls(poll_id);

ALTER TABLE ONLY polls
    ADD CONSTRAINT polls_created_by_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY polls
    ADD CONSTRAINT polls_edited_by_fkey FOREIGN KEY (edited_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY polls
    ADD CONSTRAINT polls_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY privacy_options
    ADD CONSTRAINT privacy_options_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY profile_options
    ADD CONSTRAINT profile_options_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY profiles
    ADD CONSTRAINT profiles_avatar_attachment_id_fkey FOREIGN KEY (avatar_id) REFERENCES attachments(attachment_id);

ALTER TABLE ONLY profiles
    ADD CONSTRAINT profiles_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY profiles
    ADD CONSTRAINT profiles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE ONLY read
    ADD CONSTRAINT read_item_type_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY read
    ADD CONSTRAINT read_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY revision_links
    ADD CONSTRAINT revision_links_link_id_fkey FOREIGN KEY (link_id) REFERENCES links(link_id);

ALTER TABLE ONLY revisions
    ADD CONSTRAINT revisions_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES comments(comment_id);

ALTER TABLE ONLY revisions
    ADD CONSTRAINT revisions_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY rewrite_domain_rules
    ADD CONSTRAINT rewrite_domain_rules_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES rewrite_domains(domain_id);

ALTER TABLE ONLY rewrite_domain_rules
    ADD CONSTRAINT rewrite_domain_rules_rule_id_fkey FOREIGN KEY (rule_id) REFERENCES rewrite_rules(rule_id);

ALTER TABLE ONLY role_profiles
    ADD CONSTRAINT role_profiles_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY role_profiles
    ADD CONSTRAINT role_profiles_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(role_id);

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_created_by_fkey FOREIGN KEY (created_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_edited_by_fkey FOREIGN KEY (edited_by) REFERENCES profiles(profile_id);

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_microcosm_id_fkey FOREIGN KEY (microcosm_id) REFERENCES microcosms(microcosm_id);

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY search_index
    ADD CONSTRAINT search_index_item_type_id_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY search_index
    ADD CONSTRAINT search_index_parent_item_type_id_fkey FOREIGN KEY (parent_item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY site_stats
    ADD CONSTRAINT site_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY site_options
    ADD CONSTRAINT site_options_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(site_id);

ALTER TABLE ONLY sites
    ADD CONSTRAINT sites_theme_id_fkey FOREIGN KEY (theme_id) REFERENCES themes(theme_id);

ALTER TABLE ONLY updates_latest
    ADD CONSTRAINT updates_latest_update_id_fkey FOREIGN KEY (update_id) REFERENCES updates(update_id);

ALTER TABLE ONLY attribute_values
    ADD CONSTRAINT values_attribute_id_fkey FOREIGN KEY (attribute_id) REFERENCES attribute_keys(attribute_id);

ALTER TABLE ONLY attribute_values
    ADD CONSTRAINT values_value_type_id_fkey FOREIGN KEY (value_type_id) REFERENCES value_types(value_type_id);

ALTER TABLE ONLY votes
    ADD CONSTRAINT votes_choice_id_fkey FOREIGN KEY (choice_id) REFERENCES choices(choice_id);

ALTER TABLE ONLY votes
    ADD CONSTRAINT votes_poll_id_fkey FOREIGN KEY (poll_id) REFERENCES polls(poll_id);

ALTER TABLE ONLY votes
    ADD CONSTRAINT votes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE ONLY watchers
    ADD CONSTRAINT watchers_item_type_id_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY watchers
    ADD CONSTRAINT watchers_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY ymg
    ADD CONSTRAINT ymg_item_profile_id_fkey FOREIGN KEY (item_profile_id) REFERENCES profiles(profile_id);

ALTER TABLE ONLY ymg
    ADD CONSTRAINT ymg_item_type_fkey FOREIGN KEY (item_type_id) REFERENCES item_types(item_type_id);

ALTER TABLE ONLY ymg
    ADD CONSTRAINT ymg_profile_id_fkey FOREIGN KEY (profile_id) REFERENCES profiles(profile_id);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = development, public, pg_catalog;

ALTER TABLE ONLY development.ymg DROP CONSTRAINT ymg_profile_id_fkey;
ALTER TABLE ONLY development.ymg DROP CONSTRAINT ymg_item_type_fkey;
ALTER TABLE ONLY development.ymg DROP CONSTRAINT ymg_item_profile_id_fkey;
ALTER TABLE ONLY development.watchers DROP CONSTRAINT watchers_profile_id_fkey;
ALTER TABLE ONLY development.watchers DROP CONSTRAINT watchers_item_type_id_fkey;
ALTER TABLE ONLY development.votes DROP CONSTRAINT votes_user_id_fkey;
ALTER TABLE ONLY development.votes DROP CONSTRAINT votes_poll_id_fkey;
ALTER TABLE ONLY development.votes DROP CONSTRAINT votes_choice_id_fkey;
ALTER TABLE ONLY development.attribute_values DROP CONSTRAINT values_value_type_id_fkey;
ALTER TABLE ONLY development.attribute_values DROP CONSTRAINT values_attribute_id_fkey;
ALTER TABLE ONLY development.updates_latest DROP CONSTRAINT updates_latest_update_id_fkey;
ALTER TABLE ONLY development.sites DROP CONSTRAINT sites_theme_id_fkey;
ALTER TABLE ONLY development.site_options DROP CONSTRAINT site_options_site_id_fkey;
ALTER TABLE ONLY development.site_stats DROP CONSTRAINT site_fkey;
ALTER TABLE ONLY development.search_index DROP CONSTRAINT search_index_parent_item_type_id_fkey;
ALTER TABLE ONLY development.search_index DROP CONSTRAINT search_index_item_type_id_fkey;
ALTER TABLE ONLY development.roles DROP CONSTRAINT roles_site_id_fkey;
ALTER TABLE ONLY development.roles DROP CONSTRAINT roles_microcosm_id_fkey;
ALTER TABLE ONLY development.roles DROP CONSTRAINT roles_edited_by_fkey;
ALTER TABLE ONLY development.roles DROP CONSTRAINT roles_created_by_fkey;
ALTER TABLE ONLY development.role_profiles DROP CONSTRAINT role_profiles_role_id_fkey;
ALTER TABLE ONLY development.role_profiles DROP CONSTRAINT role_profiles_profile_id_fkey;
ALTER TABLE ONLY development.rewrite_domain_rules DROP CONSTRAINT rewrite_domain_rules_rule_id_fkey;
ALTER TABLE ONLY development.rewrite_domain_rules DROP CONSTRAINT rewrite_domain_rules_domain_id_fkey;
ALTER TABLE ONLY development.revisions DROP CONSTRAINT revisions_profile_id_fkey;
ALTER TABLE ONLY development.revisions DROP CONSTRAINT revisions_comment_id_fkey;
ALTER TABLE ONLY development.revision_links DROP CONSTRAINT revision_links_link_id_fkey;
ALTER TABLE ONLY development.read DROP CONSTRAINT read_profile_id_fkey;
ALTER TABLE ONLY development.read DROP CONSTRAINT read_item_type_fkey;
ALTER TABLE ONLY development.profiles DROP CONSTRAINT profiles_user_id_fkey;
ALTER TABLE ONLY development.profiles DROP CONSTRAINT profiles_site_id_fkey;
ALTER TABLE ONLY development.profiles DROP CONSTRAINT profiles_avatar_attachment_id_fkey;
ALTER TABLE ONLY development.profile_options DROP CONSTRAINT profile_options_profile_id_fkey;
ALTER TABLE ONLY development.privacy_options DROP CONSTRAINT privacy_options_profile_id_fkey;
ALTER TABLE ONLY development.polls DROP CONSTRAINT polls_microcosm_id_fkey;
ALTER TABLE ONLY development.polls DROP CONSTRAINT polls_edited_by_fkey;
ALTER TABLE ONLY development.polls DROP CONSTRAINT polls_created_by_fkey;
ALTER TABLE ONLY development.choices DROP CONSTRAINT poll_choices_poll_id_fkey;
ALTER TABLE ONLY development.moderation_queue DROP CONSTRAINT moderation_queue_profile_id_fkey;
ALTER TABLE ONLY development.moderation_queue DROP CONSTRAINT moderation_queue_microcosm_id_fkey;
ALTER TABLE ONLY development.moderation_queue DROP CONSTRAINT moderation_queue_item_type_fkey;
ALTER TABLE ONLY development.microcosms DROP CONSTRAINT microcosms_site_id_fkey;
ALTER TABLE ONLY development.microcosms DROP CONSTRAINT microcosms_last_comment_item_type_id_fkey;
ALTER TABLE ONLY development.microcosms DROP CONSTRAINT microcosms_last_comment_created_by_fkey;
ALTER TABLE ONLY development.microcosms DROP CONSTRAINT microcosms_created_by_fkey;
ALTER TABLE ONLY development.microcosm_profile_options DROP CONSTRAINT microcosm_profile_options_profile_id_fkey;
ALTER TABLE ONLY development.microcosm_profile_options DROP CONSTRAINT microcosm_profile_options_microcosm_id_fkey;
ALTER TABLE ONLY development.microcosm_options DROP CONSTRAINT microcosm_options_microcosm_id_fkey;
ALTER TABLE ONLY development.menus DROP CONSTRAINT menus_site_id_fkey;
ALTER TABLE ONLY development.imported_items DROP CONSTRAINT imported_items_origin_id_fkey;
ALTER TABLE ONLY development.imported_items DROP CONSTRAINT imported_items_item_type_id_fkey;
ALTER TABLE ONLY development.import_origins DROP CONSTRAINT import_origins_site_id_fkey;
ALTER TABLE ONLY development.ignores DROP CONSTRAINT ignores_profile_id_fkey;
ALTER TABLE ONLY development.huddles DROP CONSTRAINT huddles_site_id_fkey;
ALTER TABLE ONLY development.huddles DROP CONSTRAINT huddles_created_by_fkey;
ALTER TABLE ONLY development.huddle_profiles DROP CONSTRAINT huddle_recipients_profile_id_fkey;
ALTER TABLE ONLY development.huddle_profiles DROP CONSTRAINT huddle_recipients_huddle_id_fkey;
ALTER TABLE ONLY development.follows DROP CONSTRAINT follows_profile_id_fkey;
ALTER TABLE ONLY development.follows DROP CONSTRAINT follows_follow_profile_id_fkey;
ALTER TABLE ONLY development.flags DROP CONSTRAINT flags_site_id_fkey;
ALTER TABLE ONLY development.flags DROP CONSTRAINT flags_parent_item_type_id_fkey;
ALTER TABLE ONLY development.flags DROP CONSTRAINT flags_item_type_id_fkey;
ALTER TABLE ONLY development.flags DROP CONSTRAINT flags_created_by_fkey;
ALTER TABLE ONLY development.events DROP CONSTRAINT events_microcosm_id_fkey;
ALTER TABLE ONLY development.disabled_roles DROP CONSTRAINT disabled_roles_role_id_fkey;
ALTER TABLE ONLY development.disabled_roles DROP CONSTRAINT disabled_roles_microcosm_id_fkey;
ALTER TABLE ONLY development.criteria DROP CONSTRAINT criteria_role_id_fkey;
ALTER TABLE ONLY development.conversations DROP CONSTRAINT conversations_microcosm_id_fkey;
ALTER TABLE ONLY development.conversations DROP CONSTRAINT conversations_edited_by_fkey;
ALTER TABLE ONLY development.conversations DROP CONSTRAINT conversations_created_by_fkey;
ALTER TABLE ONLY development.comments DROP CONSTRAINT comments_profile_id_fkey;
ALTER TABLE ONLY development.comments DROP CONSTRAINT comments_item_type_fkey;
ALTER TABLE ONLY development.bans DROP CONSTRAINT bans_user_id_fkey;
ALTER TABLE ONLY development.bans DROP CONSTRAINT bans_site_id_fkey;
ALTER TABLE ONLY development.banned_emails DROP CONSTRAINT banned_emails_user_id_fkey;
ALTER TABLE ONLY development.banned_emails DROP CONSTRAINT banned_emails_site_id_fkey;
ALTER TABLE ONLY development.attribute_keys DROP CONSTRAINT attributes_item_type_id_fkey;
ALTER TABLE ONLY development.attendees DROP CONSTRAINT attendees_state_id_fkey;
ALTER TABLE ONLY development.attendees DROP CONSTRAINT attendees_profile_id_fkey;
ALTER TABLE ONLY development.attendees DROP CONSTRAINT attendees_event_id_fkey;
ALTER TABLE ONLY development.attendees DROP CONSTRAINT attendees_edited_by_fkey;
ALTER TABLE ONLY development.attendees DROP CONSTRAINT attendees_created_by_fkey;
ALTER TABLE ONLY development.attachments DROP CONSTRAINT attachments_profile_id_fkey;
ALTER TABLE ONLY development.attachments DROP CONSTRAINT attachments_item_type_fkey;
ALTER TABLE ONLY development.attachments DROP CONSTRAINT attachments_attachment_meta_id_fkey;
ALTER TABLE ONLY development.attachment_views DROP CONSTRAINT attachment_views_attachment_id_fkey;
ALTER TABLE ONLY development.updates DROP CONSTRAINT alerts_site_id_fkey;
ALTER TABLE ONLY development.updates DROP CONSTRAINT alerts_profile_id_fkey;
ALTER TABLE ONLY development.updates DROP CONSTRAINT alerts_item_type_fkey;
ALTER TABLE ONLY development.updates DROP CONSTRAINT alerts_alerted_profile_id_fkey;
ALTER TABLE ONLY development.updates DROP CONSTRAINT alerts_alert_type_id_fkey;
ALTER TABLE ONLY development.update_options DROP CONSTRAINT alert_preferences_profile_id_fkey;
ALTER TABLE ONLY development.update_options DROP CONSTRAINT alert_preferences_alert_type_id_fkey;
ALTER TABLE ONLY development.update_options_defaults DROP CONSTRAINT alert_defaults_alert_type_id_fkey;
ALTER TABLE ONLY development.admins DROP CONSTRAINT admins_site_id_fkey;
ALTER TABLE ONLY development.admins DROP CONSTRAINT admins_profile_id_fkey;
ALTER TABLE ONLY development.activity_scores DROP CONSTRAINT activity_scores_site_id_fkey;
ALTER TABLE ONLY development.activity_scores DROP CONSTRAINT activity_scores_item_type_id_fkey;
ALTER TABLE ONLY development.access_tokens DROP CONSTRAINT access_tokens_user_id_fkey;
ALTER TABLE ONLY development.access_tokens DROP CONSTRAINT access_tokens_client_id_fkey;
DROP TRIGGER sites_flags ON development.sites;
DROP TRIGGER revisions_search_index ON development.revisions;
DROP TRIGGER profiles_search_index ON development.profiles;
DROP TRIGGER profiles_flags ON development.profiles;
DROP TRIGGER polls_flags ON development.polls;
DROP TRIGGER microcosms_search_index ON development.microcosms;
DROP TRIGGER microcosms_flags ON development.microcosms;
DROP TRIGGER huddles_search_index ON development.huddles;
DROP TRIGGER huddles_flags ON development.huddles;
DROP TRIGGER events_search_index ON development.events;
DROP TRIGGER events_flags ON development.events;
DROP TRIGGER conversations_search_index ON development.conversations;
DROP TRIGGER conversations_flags ON development.conversations;
DROP TRIGGER comments_search_index ON development.comments;
DROP TRIGGER comments_flags ON development.comments;
DROP INDEX development.watchers_profileanditemtype_idx;
DROP INDEX development.watchers_profile_id_item_type_id_item_id_idx;
DROP INDEX development.watchers_itemtypeandid_idx;
DROP INDEX development.views_itemtype_idx;
DROP INDEX development.views_item_idx;
DROP INDEX development.updates_updatetypeid_idx;
DROP INDEX development.updates_parentitemtypeandid_idx;
DROP INDEX development.updates_itemtypeandid_idx;
DROP INDEX development.updates_forprofile_idx;
DROP INDEX development.update_options_profile_id_idx;
DROP INDEX development.sites_site_id_owned_by_created_by_idx;
DROP INDEX development.searchindex_titlevector_idx;
DROP INDEX development.searchindex_siteid_idx;
DROP INDEX development.searchindex_profileid_idx;
DROP INDEX development.searchindex_parentitemtypeidandparentitemid_idx;
DROP INDEX development.searchindex_itemtypeidanditemid_idx;
DROP INDEX development.searchindex_documentvector_idx;
DROP INDEX development.revisions_profile_idx;
DROP INDEX development.revisions_comment_idx;
DROP INDEX development.revisionlinks_linkid_idx;
DROP INDEX development.read_profileitemtypeid_idx;
DROP INDEX development.read_profileitemtypeandid_idx;
DROP INDEX development.read_partial_order_idx;
DROP INDEX development.read_partial_idx;
DROP INDEX development.profiles_siteid_idx;
DROP INDEX development.polls_microcosmid_idx;
DROP INDEX development.polls_isdeleted_idx;
DROP INDEX development.parent_modified_idx;
DROP INDEX development.moderation_queue_itemtypeandid_idx;
DROP INDEX development.microcosms_microcosm_id_created_by_owned_by_idx;
DROP INDEX development.microcosms_last_comment_created_idx;
DROP INDEX development.microcosms_isdeleted_idx;
DROP INDEX development.microcosms_is_deleted_idx;
DROP INDEX development.microcosms_created_idx;
DROP INDEX development.links_url_idx;
DROP INDEX development.links_short_url_idx;
DROP INDEX development.item_types_rank_idx;
DROP INDEX development.ips_siteid_idx;
DROP INDEX development.ips_seen_idx;
DROP INDEX development.ips_profileid_idx;
DROP INDEX development.ips_ip_idx;
DROP INDEX development.ips_action_idx;
DROP INDEX development.imported_items_idx;
DROP INDEX development.ignores_profile_id_idx;
DROP INDEX development.ignores_item_type_id_item_id_idx;
DROP INDEX development.huddles_profileid_idx;
DROP INDEX development.huddles_huddleprofileid_idx;
DROP INDEX development.huddles_huddle_id_created_by_idx;
DROP INDEX development.flags_site_id_sitemap_index_idx;
DROP INDEX development.flags_partial_item_idx;
DROP INDEX development.flags_partial_idx;
DROP INDEX development.flags_parentitemtypeidandparentitemid_idx;
DROP INDEX development.flags_microcosmid_idx;
DROP INDEX development.flags_lastmodified_idx;
DROP INDEX development.flags_lastmodified2_idx;
DROP INDEX development.flags_deleted_idx;
DROP INDEX development.flags_createdby_idx;
DROP INDEX development.events_microcosmid_idx;
DROP INDEX development.events_isdeleted_idx;
DROP INDEX development.conversations_microcosmid_idx;
DROP INDEX development.conversations_isdeleted_idx;
DROP INDEX development.comments_partial_idx;
DROP INDEX development.comments_itemtypeandid_idx;
DROP INDEX development.comments_ismoderated_idx;
DROP INDEX development.comments_isdeleted_idx;
DROP INDEX development.comments_in_reply_to;
DROP INDEX development.bans_site_id_user_id_idx;
DROP INDEX development.attribute_keys_itemtypeandid_idx;
DROP INDEX development.attachments_itemtypeandid_idx;
ALTER TABLE ONLY development.ymg DROP CONSTRAINT ymg_pkey;
ALTER TABLE ONLY development.watchers DROP CONSTRAINT watchers_profile_id_item_type_id_item_id_key;
ALTER TABLE ONLY development.watchers DROP CONSTRAINT watchers_pkey;
ALTER TABLE ONLY development.votes DROP CONSTRAINT votes_pkey;
ALTER TABLE ONLY development.attribute_values DROP CONSTRAINT values_pkey;
ALTER TABLE ONLY development.value_types DROP CONSTRAINT value_types_title_key;
ALTER TABLE ONLY development.value_types DROP CONSTRAINT value_types_pkey;
ALTER TABLE ONLY development.users DROP CONSTRAINT users_pkey;
ALTER TABLE ONLY development.rewrite_rules DROP CONSTRAINT url_rewrite_id_pk;
ALTER TABLE ONLY development.updates_latest DROP CONSTRAINT updates_latest_pkey;
ALTER TABLE ONLY development.themes DROP CONSTRAINT themes_pkey;
ALTER TABLE ONLY development.sites DROP CONSTRAINT sites_subdomain_key_key;
ALTER TABLE ONLY development.sites DROP CONSTRAINT sites_domain_key;
ALTER TABLE ONLY development.site_options DROP CONSTRAINT site_options_pkey;
ALTER TABLE ONLY development.site_stats DROP CONSTRAINT site_id_pkey;
ALTER TABLE ONLY development.sites DROP CONSTRAINT site_id_pk;
ALTER TABLE ONLY development.search_index DROP CONSTRAINT search_index_pkey;
ALTER TABLE ONLY development.roles DROP CONSTRAINT roles_pkey;
ALTER TABLE ONLY development.role_profiles DROP CONSTRAINT role_profiles_pkey;
ALTER TABLE ONLY development.role_members_cache DROP CONSTRAINT role_members_cache_pkey;
ALTER TABLE ONLY development.rewrite_domains DROP CONSTRAINT rewrite_domains_pkey;
ALTER TABLE ONLY development.rewrite_domain_rules DROP CONSTRAINT rewrite_domain_rules_pkey;
ALTER TABLE ONLY development.revisions DROP CONSTRAINT revisions_pkey;
ALTER TABLE ONLY development.revisions DROP CONSTRAINT revisions_comment_id_created_key;
ALTER TABLE ONLY development.revision_links DROP CONSTRAINT revision_links_pkey;
ALTER TABLE ONLY development.read DROP CONSTRAINT read_id_pk;
ALTER TABLE ONLY development.profiles DROP CONSTRAINT profiles_site_id_user_id_key;
ALTER TABLE ONLY development.profiles DROP CONSTRAINT profiles_pkey;
ALTER TABLE ONLY development.profile_options DROP CONSTRAINT profile_options_pkey;
ALTER TABLE ONLY development.privacy_options DROP CONSTRAINT privacy_options_pkey;
ALTER TABLE ONLY development.polls DROP CONSTRAINT polls_pkey;
ALTER TABLE ONLY development.choices DROP CONSTRAINT poll_choices_pkey;
ALTER TABLE ONLY development.permissions_cache DROP CONSTRAINT permissions_cache_pkey;
ALTER TABLE ONLY development.oauth_clients DROP CONSTRAINT oauth_clients_pkey;
ALTER TABLE ONLY development.moderation_queue DROP CONSTRAINT moderation_queue_pkey;
ALTER TABLE ONLY development.microcosm_profile_options DROP CONSTRAINT microcosm_profile_options_pkey;
ALTER TABLE ONLY development.microcosm_options DROP CONSTRAINT microcosm_options_pkey;
ALTER TABLE ONLY development.microcosms DROP CONSTRAINT microcosm_id_pk;
ALTER TABLE ONLY development.metrics DROP CONSTRAINT metrics_pkey;
ALTER TABLE ONLY development.menus DROP CONSTRAINT menus_site_id_sequence_key;
ALTER TABLE ONLY development.menus DROP CONSTRAINT menus_pkey;
ALTER TABLE ONLY development.links DROP CONSTRAINT links_url_key;
ALTER TABLE ONLY development.links DROP CONSTRAINT links_short_url_key;
ALTER TABLE ONLY development.links DROP CONSTRAINT links_pkey;
ALTER TABLE ONLY development.item_types DROP CONSTRAINT item_type_id_pk;
ALTER TABLE ONLY development.ips DROP CONSTRAINT ips_pkey;
ALTER TABLE ONLY development.imported_items DROP CONSTRAINT imported_items_pkey;
ALTER TABLE ONLY development.import_origins DROP CONSTRAINT import_origins_title_key;
ALTER TABLE ONLY development.import_origins DROP CONSTRAINT import_origins_pkey;
ALTER TABLE ONLY development.ignores DROP CONSTRAINT ignores_pkey;
ALTER TABLE ONLY development.huddles DROP CONSTRAINT huddles_pkey;
ALTER TABLE ONLY development.huddle_profiles DROP CONSTRAINT huddle_recipients_pkey;
ALTER TABLE ONLY development.follows DROP CONSTRAINT follows_pkey;
ALTER TABLE ONLY development.flags DROP CONSTRAINT flags_pkey;
ALTER TABLE ONLY development.events DROP CONSTRAINT events_pkey;
ALTER TABLE ONLY development.disabled_roles DROP CONSTRAINT disabled_roles_pkey;
ALTER TABLE ONLY development.criteria DROP CONSTRAINT criteria_pkey;
ALTER TABLE ONLY development.conversations DROP CONSTRAINT conversations_pkey;
ALTER TABLE ONLY development.comments DROP CONSTRAINT comments_pkey;
ALTER TABLE ONLY development.bans DROP CONSTRAINT bans_pkey;
ALTER TABLE ONLY development.banned_emails DROP CONSTRAINT banned_emails_pk;
ALTER TABLE ONLY development.attribute_keys DROP CONSTRAINT attributes_pkey;
ALTER TABLE ONLY development.attribute_keys DROP CONSTRAINT attributes_item_type_id_item_id_name_key;
ALTER TABLE ONLY development.attendees DROP CONSTRAINT attendees_pkey;
ALTER TABLE ONLY development.attendees DROP CONSTRAINT attendees_event_id_profile_id_key;
ALTER TABLE ONLY development.attendee_state DROP CONSTRAINT attendee_state_title_key;
ALTER TABLE ONLY development.attendee_state DROP CONSTRAINT attendee_state_pkey;
ALTER TABLE ONLY development.attachment_views DROP CONSTRAINT attachment_view_id_pk;
ALTER TABLE ONLY development.attachment_meta DROP CONSTRAINT attachment_meta_id_pk;
ALTER TABLE ONLY development.attachments DROP CONSTRAINT attachment_id_pk;
ALTER TABLE ONLY development.updates DROP CONSTRAINT alerts_pkey;
ALTER TABLE ONLY development.update_types DROP CONSTRAINT alert_types_pkey;
ALTER TABLE ONLY development.update_options DROP CONSTRAINT alert_preferences_pkey;
ALTER TABLE ONLY development.update_options_defaults DROP CONSTRAINT alert_defaults_pkey;
ALTER TABLE ONLY development.admins DROP CONSTRAINT admins_site_id_key;
ALTER TABLE ONLY development.admins DROP CONSTRAINT admins_pkey;
ALTER TABLE ONLY development.access_tokens DROP CONSTRAINT access_tokens_token_value_key;
ALTER TABLE ONLY development.access_tokens DROP CONSTRAINT access_tokens_pkey;
ALTER TABLE development.ymg ALTER COLUMN ymg_id DROP DEFAULT;
ALTER TABLE development.watchers ALTER COLUMN watcher_id DROP DEFAULT;
ALTER TABLE development.users ALTER COLUMN user_id DROP DEFAULT;
ALTER TABLE development.updates ALTER COLUMN update_id DROP DEFAULT;
ALTER TABLE development.themes ALTER COLUMN theme_id DROP DEFAULT;
ALTER TABLE development.sites ALTER COLUMN site_id DROP DEFAULT;
ALTER TABLE development.roles ALTER COLUMN role_id DROP DEFAULT;
ALTER TABLE development.rewrite_rules ALTER COLUMN rule_id DROP DEFAULT;
ALTER TABLE development.rewrite_domains ALTER COLUMN domain_id DROP DEFAULT;
ALTER TABLE development.revisions ALTER COLUMN revision_id DROP DEFAULT;
ALTER TABLE development.revision_links ALTER COLUMN revlinks_id DROP DEFAULT;
ALTER TABLE development.read ALTER COLUMN read_id DROP DEFAULT;
ALTER TABLE development.profiles ALTER COLUMN profile_id DROP DEFAULT;
ALTER TABLE development.polls ALTER COLUMN poll_id DROP DEFAULT;
ALTER TABLE development.oauth_clients ALTER COLUMN client_id DROP DEFAULT;
ALTER TABLE development.moderation_queue ALTER COLUMN moderation_queue_id DROP DEFAULT;
ALTER TABLE development.microcosms ALTER COLUMN microcosm_id DROP DEFAULT;
ALTER TABLE development.microcosm_profile_options ALTER COLUMN microcosm_profile_option_id DROP DEFAULT;
ALTER TABLE development.menus ALTER COLUMN menu_id DROP DEFAULT;
ALTER TABLE development.links ALTER COLUMN link_id DROP DEFAULT;
ALTER TABLE development.import_origins ALTER COLUMN origin_id DROP DEFAULT;
ALTER TABLE development.huddles ALTER COLUMN huddle_id DROP DEFAULT;
ALTER TABLE development.events ALTER COLUMN event_id DROP DEFAULT;
ALTER TABLE development.criteria ALTER COLUMN criteria_id DROP DEFAULT;
ALTER TABLE development.conversations ALTER COLUMN conversation_id DROP DEFAULT;
ALTER TABLE development.comments ALTER COLUMN comment_id DROP DEFAULT;
ALTER TABLE development.choices ALTER COLUMN choice_id DROP DEFAULT;
ALTER TABLE development.bans ALTER COLUMN ban_id DROP DEFAULT;
ALTER TABLE development.attribute_keys ALTER COLUMN attribute_id DROP DEFAULT;
ALTER TABLE development.attendees ALTER COLUMN attendee_id DROP DEFAULT;
ALTER TABLE development.attachments ALTER COLUMN attachment_id DROP DEFAULT;
ALTER TABLE development.attachment_meta ALTER COLUMN attachment_meta_id DROP DEFAULT;
ALTER TABLE development.access_tokens ALTER COLUMN access_token_id DROP DEFAULT;
DROP SEQUENCE development.ymg_ymg_id_seq;
DROP TABLE development.ymg;
DROP SEQUENCE development.watchers_watcher_id_seq;
DROP TABLE development.watchers;
DROP TABLE development.votes;
DROP TABLE development.views;
DROP TABLE development.value_types;
DROP SEQUENCE development.users_user_id_seq;
DROP SEQUENCE development.url_rewrites_url_rewrite_id_seq;
DROP TABLE development.updates_latest;
DROP TABLE development.update_types;
DROP TABLE development.update_options_defaults;
DROP TABLE development.update_options;
DROP SEQUENCE development.themes_theme_id_seq;
DROP TABLE development.themes;
DROP SEQUENCE development.sites_site_id_seq;
DROP TABLE development.sites;
DROP TABLE development.site_stats;
DROP TABLE development.site_options;
DROP TABLE development.search_index;
DROP SEQUENCE development.roles_role_id_seq;
DROP TABLE development.roles;
DROP TABLE development.role_profiles;
DROP TABLE development.role_members_cache;
DROP TABLE development.rewrite_rules;
DROP SEQUENCE development.rewrite_domains_domain_id_seq;
DROP TABLE development.rewrite_domains;
DROP TABLE development.rewrite_domain_rules;
DROP SEQUENCE development.revisions_revision_id_seq;
DROP TABLE development.revisions;
DROP SEQUENCE development.revision_links_revlinks_id_seq;
DROP TABLE development.revision_links;
DROP SEQUENCE development.read_read_id_seq;
DROP TABLE development.read;
DROP SEQUENCE development.profiles_profile_id_seq;
DROP TABLE development.profile_options;
DROP VIEW development.profile_filter;
DROP TABLE development.users;
DROP TABLE development.profiles;
DROP TABLE development.privacy_options;
DROP SEQUENCE development.polls_poll_id_seq;
DROP TABLE development.polls;
DROP TABLE development.platform_options;
DROP TABLE development.permissions_cache;
DROP SEQUENCE development.oauth_clients_client_id_seq;
DROP TABLE development.oauth_clients;
DROP SEQUENCE development.moderation_queue_moderation_queue_id_seq;
DROP TABLE development.moderation_queue;
DROP SEQUENCE development.microcosms_microcosm_id_seq;
DROP TABLE development.microcosms;
DROP SEQUENCE development.microcosm_profile_options_microcosm_profile_option_id_seq;
DROP TABLE development.microcosm_profile_options;
DROP TABLE development.microcosm_options;
DROP TABLE development.metrics;
DROP SEQUENCE development.menus_menu_id_seq;
DROP TABLE development.menus;
DROP SEQUENCE development.links_link_id_seq;
DROP TABLE development.links;
DROP TABLE development.item_types;
DROP TABLE development.ips;
DROP TABLE development.imported_items;
DROP SEQUENCE development.import_origins_origin_id_seq;
DROP TABLE development.import_origins;
DROP TABLE development.ignores;
DROP SEQUENCE development.huddles_huddle_id_seq;
DROP TABLE development.huddles;
DROP TABLE development.huddle_profiles;
DROP TABLE development.follows;
DROP TABLE development.flags;
DROP SEQUENCE development.events_event_id_seq;
DROP TABLE development.events;
DROP TABLE development.disabled_roles;
DROP SEQUENCE development.criteria_criteria_id_seq;
DROP TABLE development.criteria;
DROP SEQUENCE development.conversations_conversation_id_seq;
DROP TABLE development.conversations;
DROP SEQUENCE development.comments_comment_id_seq;
DROP TABLE development.comments;
DROP SEQUENCE development.choices_choice_id_seq;
DROP TABLE development.choices;
DROP SEQUENCE development.bans_ban_id_seq;
DROP TABLE development.bans;
DROP TABLE development.banned_emails;
DROP SEQUENCE development.attributes_attribute_id_seq;
DROP TABLE development.attribute_values;
DROP TABLE development.attribute_keys;
DROP SEQUENCE development.attendees_attendee_id_seq;
DROP TABLE development.attendees;
DROP TABLE development.attendee_state;
DROP SEQUENCE development.attachments_attachment_id_seq;
DROP TABLE development.attachments;
DROP TABLE development.attachment_views;
DROP SEQUENCE development.attachment_meta_attachment_meta_id_seq;
DROP TABLE development.attachment_meta;
DROP SEQUENCE development.alerts_alert_id_seq;
DROP TABLE development.updates;
DROP TABLE development.admins;
DROP TABLE development.activity_scores;
DROP SEQUENCE development.access_tokens_access_token_id_seq;
DROP TABLE development.access_tokens;
DROP FUNCTION development.update_sites_flags();
DROP FUNCTION development.update_revisions_search_index();
DROP FUNCTION development.update_profiles_search_index();
DROP FUNCTION development.update_profiles_flags();
DROP FUNCTION development.update_polls_flags();
DROP FUNCTION development.update_microcosms_search_index();
DROP FUNCTION development.update_microcosms_flags();
DROP FUNCTION development.update_huddles_search_index();
DROP FUNCTION development.update_huddles_flags();
DROP FUNCTION development.update_events_search_index();
DROP FUNCTION development.update_events_flags();
DROP FUNCTION development.update_conversations_search_index();
DROP FUNCTION development.update_conversations_flags();
DROP FUNCTION development.update_comments_search_index();
DROP FUNCTION development.update_comments_flags();
DROP FUNCTION development.last_read_time(in_item_type_id bigint, in_item_id bigint, in_profile_id bigint);
DROP FUNCTION development.is_deleted(initemtypeid bigint, initemid bigint);
DROP FUNCTION development.is_banned(site_id bigint, microcosm_id bigint, profile_id bigint);
DROP FUNCTION development.is_attending(in_event_id bigint, in_profile_id bigint);
DROP FUNCTION development.has_unread(in_item_type_id bigint, in_item_id bigint, in_profile_id bigint);
DROP FUNCTION development.get_role_profiles(site_id bigint, role_id bigint);
DROP FUNCTION development.get_role_profile(insiteid bigint, inroleid bigint, inprofileid bigint);
DROP FUNCTION development.get_microcosm_roles(insiteid bigint, inmicrocosmid bigint);
DROP FUNCTION development.get_microcosm_permissions_for_profile(insiteid bigint, inmicrocosmid bigint, inprofileid bigint);
DROP FUNCTION development.get_effective_permissions(insiteid bigint, inmicrocosmid bigint, initemtypeid bigint, initemid bigint, inprofileid bigint);
DROP FUNCTION development.get_communication_options(in_site_id integer, in_item_id integer, in_item_type_id integer, in_profile_id integer, in_update_type_id integer);
DROP FUNCTION development.create_owned_site(in_title character varying, in_subdomain_key character varying, in_theme_id bigint, in_user_id bigint, in_profile_name character varying, in_avatar_id bigint, in_avatar_url character varying, in_domain character varying, in_description text, in_logo_url character varying, in_background_url character varying, in_background_position character varying, in_background_color character varying, in_link_color character varying, in_ga_web_property_id character varying);
DROP TYPE development.effective_permissions;
DROP SCHEMA development;
