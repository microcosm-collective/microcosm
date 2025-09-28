-- Delete content and banning spammer
-- with the profile ID of 999999

-- Optional. Disable psql's pager with:
-- \pset pager off

-- Verify we have the right person:
SELECT * FROM profiles WHERE profile_id = 999999;

-- Show any profiles for other sites:
SELECT p1.* FROM profiles AS p1
JOIN profiles AS p2 ON p1.user_id = p2.user_id AND p2.profile_id = 999999;

-- Show comment count (typically 1):
SELECT COUNT(*) FROM comments WHERE profile_id = 999999;

-- Show email address
SELECT email FROM users AS u
JOIN profiles AS p ON u.user_id = p.user_id
WHERE p.profile_id = 999999;


-- Clean up:
UPDATE conversations SET is_deleted = TRUE WHERE created_by = 999999;

UPDATE events SET is_deleted = TRUE WHERE created_by = 999999;

UPDATE comments SET is_deleted = TRUE WHERE profile_id = 999999;

DELETE FROM huddle_profiles WHERE huddle_id IN (
	SELECT huddle_id FROM huddle_profiles WHERE profile_id = 999999
);

INSERT INTO bans (
	site_id,
	user_id,
	created,
	display_reason,
	admin_reason
) VALUES (
	(SELECT site_id FROM profiles WHERE profile_id = 999999),
	(SELECT user_id FROM profiles WHERE profile_id = 999999),
	NOW(),
	'Spammer',
	'Spammer'
);

DELETE FROM role_members_cache WHERE profile_id = 999999;

DELETE FROM permissions_cache WHERE profile_id = 999999;

DELETE FROM access_tokens WHERE user_id IN (
    SELECT user_id FROM profiles WHERE profile_id = 999999
);
