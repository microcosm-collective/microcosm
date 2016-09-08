-- Will swap the emails of two users, which is useful if someone once used an email
-- address, and then has signed up with a new email and actually wants to use that
-- to access their original account

SELECT p.profile_id
      ,p.profile_name
      ,u.user_id
      ,u.email
      ,u.canonical_email
  FROM profiles p
      ,users u
 WHERE p.user_id = u.user_id
   AND p.site_id = 234
   AND p.profile_name IN ('user12345','profile1');

-- This is in a transaction, does nothing unless we run the parts without the ROLLBACK
BEGIN;

UPDATE users
   SET email = 'profile1@gmail.com'
      ,canonical_email = canonical_email('profile1@gmail.com')
 WHERE user_id = 27006; -- user ID of user12345

UPDATE users
   SET email = 'user12345@gmail.com'
      ,canonical_email = canonical_email('user12345@gmail.com')
 WHERE user_id = 61709; -- user ID of profile1

ROLLBACK;
