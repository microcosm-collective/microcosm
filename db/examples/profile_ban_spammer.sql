-- deleting content and banning spammer:
-- with the profile ID of 94450
-- for the site ID 234

UPDATE conversations SET is_deleted = TRUE WHERE created_by = 94450;

UPDATE events SET is_deleted = TRUE WHERE created_by = 94450;

UPDATE comments SET is_deleted = TRUE WHERE profile_id = 94450;

DELETE FROM huddle_profiles WHERE huddle_id IN (
	SELECT huddle_id FROM huddle_profiles WHERE profile_id = 94450
);

INSERT INTO bans (
	site_id,
	user_id,
	created,
	display_reason,
	admin_reason
) VALUES (
	234,
	(SELECT user_id FROM profiles WHERE profile_id = 94450),
	NOW(),
	'Spammer',
	'Spammer'
);

DELETE FROM role_members_cache WHERE profile_id = 94450;

DELETE FROM permissions_cache WHERE profile_id = 94450;

SELECT email
  FROM users u
  JOIN profiles p ON u.user_id = p.user_id
 WHERE p.profile_id = 94450;
