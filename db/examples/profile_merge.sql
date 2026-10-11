-- Safe profile merge script (PostgreSQL)
-- Set these two IDs before running:
--   source_profile_id: profile to remove
--   target_profile_id: profile to keep

-- !!! THIS IS UNTESTED !!!

BEGIN;

DO $$
DECLARE
    source_profile_id bigint := 99999999;
    target_profile_id bigint := 11111111;

    source_user_id bigint;
    target_user_id bigint;
    source_site_id bigint;
    target_site_id bigint;
BEGIN
    IF source_profile_id = target_profile_id THEN
        RAISE EXCEPTION 'source_profile_id and target_profile_id are the same (%).', source_profile_id;
    END IF;

    SELECT user_id, site_id
      INTO source_user_id, source_site_id
      FROM profiles
     WHERE profile_id = source_profile_id
     FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Source profile % does not exist.', source_profile_id;
    END IF;

    SELECT user_id, site_id
      INTO target_user_id, target_site_id
      FROM profiles
     WHERE profile_id = target_profile_id
     FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Target profile % does not exist.', target_profile_id;
    END IF;

    IF source_site_id <> target_site_id THEN
        RAISE EXCEPTION 'Cross-site merge not allowed: source site %, target site %.',
            source_site_id, target_site_id;
    END IF;

    -- 1) Tables with uniqueness risk: insert/merge then delete source rows

    INSERT INTO admins (site_id, profile_id, created)
    SELECT site_id, target_profile_id, created
      FROM admins
     WHERE profile_id = source_profile_id
    ON CONFLICT (site_id, profile_id) DO NOTHING;
    DELETE FROM admins WHERE profile_id = source_profile_id;

    INSERT INTO huddle_profiles (huddle_id, profile_id)
    SELECT huddle_id, target_profile_id
      FROM huddle_profiles
     WHERE profile_id = source_profile_id
    ON CONFLICT (huddle_id, profile_id) DO NOTHING;
    DELETE FROM huddle_profiles WHERE profile_id = source_profile_id;

    INSERT INTO role_profiles (role_id, profile_id)
    SELECT role_id, target_profile_id
      FROM role_profiles
     WHERE profile_id = source_profile_id
    ON CONFLICT (role_id, profile_id) DO NOTHING;
    DELETE FROM role_profiles WHERE profile_id = source_profile_id;

    INSERT INTO watchers (profile_id, item_type_id, item_id, last_notified, send_email, send_sms)
    SELECT target_profile_id, item_type_id, item_id, last_notified, send_email, send_sms
      FROM watchers
     WHERE profile_id = source_profile_id
    ON CONFLICT (profile_id, item_type_id, item_id) DO NOTHING;
    DELETE FROM watchers WHERE profile_id = source_profile_id;

    INSERT INTO ignores (profile_id, item_type_id, item_id)
    SELECT target_profile_id, item_type_id, item_id
      FROM ignores
     WHERE profile_id = source_profile_id
    ON CONFLICT (profile_id, item_type_id, item_id) DO NOTHING;
    DELETE FROM ignores WHERE profile_id = source_profile_id;

    INSERT INTO update_options (profile_id, update_type_id, send_email, send_sms)
    SELECT target_profile_id, update_type_id, send_email, send_sms
      FROM update_options
     WHERE profile_id = source_profile_id
    ON CONFLICT (profile_id, update_type_id) DO UPDATE
      SET send_email = update_options.send_email OR EXCLUDED.send_email,
          send_sms   = update_options.send_sms   OR EXCLUDED.send_sms;
    DELETE FROM update_options WHERE profile_id = source_profile_id;

    -- follows has source in either column; rewrite both columns into target
    INSERT INTO follows (profile_id, follow_profile_id, created)
    SELECT CASE WHEN profile_id = source_profile_id THEN target_profile_id ELSE profile_id END,
           CASE WHEN follow_profile_id = source_profile_id THEN target_profile_id ELSE follow_profile_id END,
           MIN(created)
      FROM follows
     WHERE profile_id = source_profile_id
        OR follow_profile_id = source_profile_id
     GROUP BY 1,2
    ON CONFLICT (profile_id, follow_profile_id) DO UPDATE
      SET created = LEAST(follows.created, EXCLUDED.created);

    DELETE FROM follows
     WHERE profile_id = source_profile_id
        OR follow_profile_id = source_profile_id;

    -- attendees has UNIQUE(event_id, profile_id): drop colliding source attendee rows first
    DELETE FROM attendees a
     USING attendees t
     WHERE a.profile_id = source_profile_id
       AND t.profile_id = target_profile_id
       AND a.event_id   = t.event_id;

    -- 2) Simple FK updates

    UPDATE updates                   SET for_profile_id          = target_profile_id WHERE for_profile_id          = source_profile_id;
    UPDATE updates                   SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE attachments               SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE attendees                 SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE attendees                 SET edited_by               = target_profile_id WHERE edited_by               = source_profile_id;
    UPDATE attendees                 SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE comments                  SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE conversations             SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE conversations             SET edited_by               = target_profile_id WHERE edited_by               = source_profile_id;
    UPDATE flags                     SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE huddles                   SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE microcosm_profile_options SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE microcosms                SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE microcosms                SET last_comment_created_by = target_profile_id WHERE last_comment_created_by = source_profile_id;
    UPDATE moderation_queue          SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE polls                     SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE polls                     SET edited_by               = target_profile_id WHERE edited_by               = source_profile_id;
    UPDATE revisions                 SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE roles                     SET created_by              = target_profile_id WHERE created_by              = source_profile_id;
    UPDATE roles                     SET edited_by               = target_profile_id WHERE edited_by               = source_profile_id;
    UPDATE ymg                       SET item_profile_id         = target_profile_id WHERE item_profile_id         = source_profile_id;
    UPDATE ymg                       SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE ips                       SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE search_index              SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;
    UPDATE read                      SET profile_id              = target_profile_id WHERE profile_id              = source_profile_id;

    -- Remap references where item_type_id=3 means item_id is profile_id
    -- 1) parent-item references
    UPDATE updates
       SET parent_item_id = target_profile_id
     WHERE parent_item_type_id = 3
       AND parent_item_id = source_profile_id;

    UPDATE flags
       SET parent_item_id = target_profile_id
     WHERE parent_item_type_id = 3
       AND parent_item_id = source_profile_id;

    UPDATE search_index
       SET parent_item_id = target_profile_id
     WHERE parent_item_type_id = 3
       AND parent_item_id = source_profile_id;

    -- 2) direct item references with collision-safe handling where needed

    -- flags: PK(item_type_id,item_id)
    DELETE FROM flags f
     WHERE f.item_type_id = 3
       AND f.item_id = source_profile_id
       AND EXISTS (
           SELECT 1
             FROM flags t
            WHERE t.item_type_id = 3
              AND t.item_id = target_profile_id
       );
    UPDATE flags
       SET item_id = target_profile_id
     WHERE item_type_id = 3
       AND item_id = source_profile_id;

    -- search_index: PK(item_type_id,item_id)
    DELETE FROM search_index s
     WHERE s.item_type_id = 3
       AND s.item_id = source_profile_id
       AND EXISTS (
           SELECT 1
             FROM search_index t
            WHERE t.item_type_id = 3
              AND t.item_id = target_profile_id
       );
    UPDATE search_index
       SET item_id = target_profile_id
     WHERE item_type_id = 3
       AND item_id = source_profile_id;

    -- ignores: PK(profile_id,item_type_id,item_id)
    DELETE FROM ignores i
     USING ignores t
     WHERE i.item_type_id = 3
       AND i.item_id = source_profile_id
       AND t.profile_id = i.profile_id
       AND t.item_type_id = 3
       AND t.item_id = target_profile_id;
    UPDATE ignores
       SET item_id = target_profile_id
     WHERE item_type_id = 3
       AND item_id = source_profile_id;

    -- watchers: UNIQUE(profile_id,item_type_id,item_id)
    DELETE FROM watchers w
     USING watchers t
     WHERE w.item_type_id = 3
       AND w.item_id = source_profile_id
       AND t.profile_id = w.profile_id
       AND t.item_type_id = 3
       AND t.item_id = target_profile_id;
    UPDATE watchers
       SET item_id = target_profile_id
     WHERE item_type_id = 3
       AND item_id = source_profile_id;

    -- permissions_cache: PK(site_id,profile_id,item_type_id,item_id)
    DELETE FROM permissions_cache p
     USING permissions_cache t
     WHERE p.item_type_id = 3
       AND p.item_id = source_profile_id
       AND t.site_id = p.site_id
       AND t.profile_id = p.profile_id
       AND t.item_type_id = 3
       AND t.item_id = target_profile_id;
    UPDATE permissions_cache
       SET item_id = target_profile_id
     WHERE item_type_id = 3
       AND item_id = source_profile_id;

    -- ips: PK(site_id,item_type_id,item_id,profile_id,seen)
    DELETE FROM ips i
     USING ips t
     WHERE i.item_type_id = 3
       AND i.item_id = source_profile_id
       AND t.site_id = i.site_id
       AND t.profile_id = i.profile_id
       AND t.seen = i.seen
       AND t.item_type_id = 3
       AND t.item_id = target_profile_id;
    UPDATE ips
       SET item_id = target_profile_id
     WHERE item_type_id = 3
       AND item_id = source_profile_id;

    -- no uniqueness issue expected
    UPDATE updates          SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;
    UPDATE comments         SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;
    UPDATE attachments      SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;
    UPDATE moderation_queue SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;
    UPDATE read             SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;
    UPDATE views            SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;
    UPDATE ymg              SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;
    UPDATE activity_scores  SET item_id = target_profile_id WHERE item_type_id = 3 AND item_id = source_profile_id;


    -- one-row-per-profile tables: keep target row, remove source row
    DELETE FROM privacy_options WHERE profile_id = source_profile_id;
    DELETE FROM profile_options WHERE profile_id = source_profile_id;

    -- housekeeping
    DELETE FROM ignores WHERE item_type_id = 3 AND item_id = source_profile_id;
    DELETE FROM permissions_cache  WHERE profile_id IN (source_profile_id, target_profile_id);
    DELETE FROM role_members_cache WHERE profile_id IN (source_profile_id, target_profile_id);

    -- remove merged profile
    DELETE FROM profiles WHERE profile_id = source_profile_id;

    -- optional: delete source user only if no profiles remain
    -- (left disabled to avoid accidental user-level data loss)
    -- IF source_user_id <> target_user_id
    --    AND (SELECT COUNT(*) FROM profiles WHERE user_id = source_user_id) = 0 THEN
    --     DELETE FROM access_tokens WHERE user_id = source_user_id;
    --     DELETE FROM users         WHERE user_id = source_user_id;
    -- END IF;

    -- recompute target counts
    UPDATE profiles
       SET comment_count = 0,
           item_count = 0
     WHERE profile_id = target_profile_id;

    UPDATE profiles p
       SET comment_count = c.comment_count
      FROM (
            SELECT created_by AS profile_id, COUNT(*) AS comment_count
              FROM flags
             WHERE created_by = target_profile_id
               AND item_type_id = 4
               AND microcosm_is_deleted IS NOT TRUE
               AND microcosm_is_moderated IS NOT TRUE
               AND parent_is_deleted IS NOT TRUE
               AND parent_is_moderated IS NOT TRUE
               AND item_is_deleted IS NOT TRUE
               AND item_is_moderated IS NOT TRUE
             GROUP BY created_by
           ) c
     WHERE p.profile_id = c.profile_id;

    UPDATE profiles p
       SET item_count = c.item_count
      FROM (
            SELECT created_by AS profile_id, COUNT(*) AS item_count
              FROM flags
             WHERE created_by = target_profile_id
               AND item_type_id IN (6,9)
               AND microcosm_is_deleted IS NOT TRUE
               AND microcosm_is_moderated IS NOT TRUE
               AND parent_is_deleted IS NOT TRUE
               AND parent_is_moderated IS NOT TRUE
               AND item_is_deleted IS NOT TRUE
               AND item_is_moderated IS NOT TRUE
             GROUP BY created_by
           ) c
     WHERE p.profile_id = c.profile_id;

END $$;

ROLLBACK;
