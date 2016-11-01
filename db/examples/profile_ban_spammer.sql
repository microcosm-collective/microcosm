﻿-- deleting content and banning spammer:
-- with the profile ID of 110714
-- for the site ID 234

UPDATE conversations SET is_deleted = TRUE WHERE created_by = 110714;

UPDATE events SET is_deleted = TRUE WHERE created_by = 110714;

UPDATE comments SET is_deleted = TRUE WHERE profile_id = 110714;

DELETE FROM huddle_profiles WHERE huddle_id IN (
	SELECT huddle_id FROM huddle_profiles WHERE profile_id = 110714
);

INSERT INTO bans (
	site_id,
	user_id,
	created,
	display_reason,
	admin_reason
) VALUES (
	234,
	(SELECT user_id FROM profiles WHERE profile_id = 110714),
	NOW(),
	'Spammer',
	'Spammer'
);

DELETE FROM role_members_cache WHERE profile_id = 110714;

DELETE FROM permissions_cache WHERE profile_id = 110714;

DELETE FROM access_tokens WHERE user_id IN (
    SELECT user_id FROM profiles WHERE profile_id = 110714
);

SELECT email
  FROM users u
  JOIN profiles p ON u.user_id = p.user_id
 WHERE p.profile_id = 110714;
