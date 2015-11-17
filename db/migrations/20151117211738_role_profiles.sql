
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
              ,rr.is_superuser
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
                  ,r.is_moderator_role AS is_superuser
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
              ,BOOL_OR(c.is_superuser OR result.is_superuser)
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    END LOOP;

    RETURN result;

END;
$BODY$
  LANGUAGE plpgsql STABLE
  COST 100;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_effective_permissions(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0,
    initemtypeid bigint DEFAULT 0,
    initemid bigint DEFAULT 0,
    inprofileid bigint DEFAULT 0)
  RETURNS effective_permissions AS
$BODY$
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

        -- We lookup moderators as part of get_microcosm_permissions_for_profile
        -- and so no longer do that here.

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
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

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
                  ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
              FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
              INTO result.can_create
                  ,result.can_read
                  ,result.can_update
                  ,result.can_delete
                  ,result.can_close_own
                  ,result.can_open_own
                  ,result.can_read_others
                  ,result.is_superuser;
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
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    WHEN 7 THEN -- Poll
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    WHEN 8 THEN -- Article
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    WHEN 9 THEN -- Event
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    WHEN 10 THEN -- Question
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    WHEN 11 THEN -- Classified
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    WHEN 12 THEN -- Album
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

    WHEN 13 THEN -- Attendee
        -- Fetch Microcosm permissions
        SELECT (rr.can_create OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_read OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_update OR result.is_site_owner OR result.is_superuser OR result.is_owner)
              ,(rr.can_delete OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_close_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_open_own OR result.is_site_owner OR result.is_superuser)
              ,(rr.can_read_others OR result.is_site_owner OR result.is_superuser)
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

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
              ,(rr.is_superuser OR result.is_site_owner OR result.is_superuser)
          FROM get_microcosm_permissions_for_profile(insiteid, mid, inprofileid) as rr
          INTO result.can_create
              ,result.can_read
              ,result.can_update
              ,result.can_delete
              ,result.can_close_own
              ,result.can_open_own
              ,result.can_read_others
              ,result.is_superuser;

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
$BODY$
  LANGUAGE plpgsql VOLATILE
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_effective_permissions(
    insiteid bigint DEFAULT 0,
    inmicrocosmid bigint DEFAULT 0,
    initemtypeid bigint DEFAULT 0,
    initemid bigint DEFAULT 0,
    inprofileid bigint DEFAULT 0)
  RETURNS effective_permissions AS
$BODY$
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
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
-- +goose StatementEnd

TRUNCATE role_members_cache;
TRUNCATE permissions_cache;
