-- Delete profile is actually merge... we move all content from the profile to
-- be deleted to another profile that will now own the content.

-- profile to delete = 81859
-- user_id of profile to delete 540
--
-- profile to own all content = 47687

-- get the deleted user id, in our case 47687
SELECT profile_id
  FROM profiles
 WHERE site_id = 234
   AND profile_name = 'deleted';

-- or if it is a merge, fine the two profiles affected
SELECT p.profile_id
      ,p.profile_name
      ,u.user_id
      ,u.email
      ,u.canonical_email
  FROM profiles p
      ,users u
 WHERE p.user_id = u.user_id
   AND p.site_id = 234
   AND p.profile_name IN ('keanulooser');

BEGIN;

UPDATE flags
   SET created_by = 47687
 WHERE created_by = 81859;
 
UPDATE search_index
   SET profile_id = 47687
 WHERE profile_id = 81859;

UPDATE revisions
   SET profile_id = 47687
 WHERE profile_id = 81859;

UPDATE comments
   SET profile_id = 47687
 WHERE profile_id = 81859;

UPDATE conversations
   SET created_by = 47687
 WHERE created_by = 81859;

UPDATE conversations
   SET edited_by = 47687
 WHERE edited_by = 81859;

UPDATE attendees
   SET created_by = 47687
 WHERE created_by = 81859;

UPDATE attendees
   SET edited_by = 47687
 WHERE edited_by = 81859;

UPDATE attendees
   SET attendee_id = 47687
 WHERE attendee_id = 81859;

UPDATE attendees
   SET profile_id = 47687
 WHERE profile_id = 81859;
 
UPDATE events
   SET created_by = 47687
 WHERE created_by = 81859;

DELETE FROM updates
 WHERE for_profile_id = 81859;

UPDATE updates
   SET created_by = 47687
 WHERE created_by = 81859;

DELETE FROM update_options
 WHERE profile_id = 81859;

DELETE FROM profile_options
 WHERE profile_id = 81859;

DELETE FROM read
 WHERE profile_id = 81859;

DELETE FROM role_profiles
 WHERE profile_id = 81859;

DELETE FROM huddle_profiles
 WHERE profile_id = 81859;

UPDATE huddles
   SET created_by = 47687
 WHERE created_by = 81859;

DELETE FROM watchers
 WHERE profile_id = 81859;

UPDATE profiles
   SET avatar_id = NULL
 WHERE profile_id = 81859
   AND avatar_id IS NOT NULL;

UPDATE attachments
   SET profile_id = 47687
 WHERE profile_id = 81859;

DELETE FROM follows
 WHERE profile_id = 81859;

UPDATE microcosms
   SET created_by = 47687
 WHERE created_by = 81859;

DELETE FROM ignores
 WHERE profile_id = 81859;

DELETE FROM ignores
 WHERE item_type_id = 3
   AND item_id = 81859;


DELETE FROM profiles
 WHERE profile_id = 81859;

-- If this was the users only profile, delete the user and access token
DO
$do$
BEGIN
IF (SELECT COUNT(*) FROM profiles WHERE user_id = 540) = 1 THEN
	DELETE FROM access_tokens WHERE user_id = 540;
	DELETE FROM users WHERE user_id = 540;
END IF;
END
$do$;

-- Update the receiving users count of comments and items
UPDATE profiles
   SET comment_count = 0
      ,item_count = 0
 WHERE profile_id = 47687;

UPDATE profiles AS p
   SET comment_count = c.comment_count
  FROM (
 SELECT created_by AS profile_id
       ,COUNT(*) AS comment_count
   FROM flags
  WHERE created_by = 47687
    AND item_type_id = 4
    AND microcosm_is_deleted IS NOT TRUE
    AND microcosm_is_moderated IS NOT TRUE
    AND parent_is_deleted IS NOT TRUE
    AND parent_is_moderated IS NOT TRUE
    AND item_is_deleted IS NOT TRUE
    AND item_is_moderated IS NOT TRUE
  GROUP BY created_by
       ) AS c
 WHERE p.profile_id = c.profile_id;

UPDATE profiles AS p
   SET item_count = c.item_count
  FROM (
 SELECT created_by AS profile_id
       ,COUNT(*) AS item_count
   FROM flags
  WHERE created_by = 47687
    AND item_type_id IN (6,9)
    AND microcosm_is_deleted IS NOT TRUE
    AND microcosm_is_moderated IS NOT TRUE
    AND parent_is_deleted IS NOT TRUE
    AND parent_is_moderated IS NOT TRUE
    AND item_is_deleted IS NOT TRUE
    AND item_is_moderated IS NOT TRUE
  GROUP BY created_by
       ) AS c
 WHERE p.profile_id = c.profile_id;

COMMIT;
