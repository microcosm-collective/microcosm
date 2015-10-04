-- site 999 will be deleted

BEGIN;

UPDATE sites
   SET is_deleted = TRUE
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM permissions_cache
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM role_members_cache
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM flags
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);
 
DELETE FROM search_index
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM revision_links
 WHERE revision_id IN (
           SELECT revision_id
             FROM revisions
            WHERE profile_id IN (
                      SELECT profile_id
                        FROM profiles
                       WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
                  )
       );

DELETE FROM revisions
 WHERE profile_id IN (
           SELECT profile_id
             FROM profiles
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM comments
 WHERE profile_id IN (
           SELECT profile_id
             FROM profiles
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM conversations
 WHERE created_by IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM attendees
 WHERE created_by IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       )
    OR attendee_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM events
 WHERE created_by IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM updates_latest
 WHERE update_id IN (
           SELECT update_id
             FROM updates
            WHERE for_profile_id IN (
                      SELECT profile_id 
                        FROM profiles 
                       WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
                  )
       );

DELETE FROM updates
 WHERE for_profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM update_options
 WHERE profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM profile_options
 WHERE profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM read
 WHERE profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM criteria
 WHERE role_id IN (
           SELECT role_id 
             FROM roles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM role_profiles
 WHERE role_id IN (
           SELECT role_id 
             FROM roles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM roles
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM microcosms
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM huddle_profiles
 WHERE profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM huddles
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM watchers
 WHERE profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

UPDATE profiles
   SET avatar_id = NULL
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
   AND avatar_id IS NOT NULL;

DELETE FROM attachments
 WHERE profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM ips
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM menus
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM follows
 WHERE profile_id IN (
           SELECT profile_id 
             FROM profiles 
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)
       );

DELETE FROM banned_emails
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM bans
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM imported_items
 WHERE origin_id IN (
           SELECT origin_id
             FROM import_origins
            WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999)         
       );

DELETE FROM import_origins
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM admins
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM site_options
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM activity_scores
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM site_stats
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM ignores
 WHERE profile_id IN (SELECT profile_id FROM profiles WHERE site_id = 999);

DELETE FROM profiles
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

DELETE FROM sites
 WHERE site_id IN (SELECT site_id FROM sites WHERE site_id = 999);

ROLLBACK;

-- Database Cleanup

DELETE FROM attachment_meta
 WHERE attachment_meta_id NOT IN (SELECT DISTINCT(attachment_meta_id) FROM attachments);

DELETE FROM links
 WHERE link_id NOT IN (SELECT DISTINCT(link_id) FROM revision_links);

DELETE FROM access_tokens WHERE user_id NOT IN (SELECT DISTINCT(user_id) FROM profiles);

DELETE FROM users WHERE user_id NOT IN (SELECT DISTINCT(user_id) FROM profiles);

DELETE FROM attribute_values WHERE attribute_id IN (
	SELECT attribute_id FROM attribute_keys WHERE item_type_id = 3 AND item_id NOT IN (SELECT profile_id FROM profiles));

DELETE FROM attribute_keys WHERE item_type_id = 3 AND item_id NOT IN (SELECT profile_id FROM profiles);

-- Database Compress

UPDATE revisions SET html = NULL;
TRUNCATE views;
TRUNCATE attachment_views;
TRUNCATE oauth_clients CASCADE;
TRUNCATE metrics;

VACUUM FULL;
VACUUM ANALYZE;

