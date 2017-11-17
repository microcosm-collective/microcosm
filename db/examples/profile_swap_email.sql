-- Will swap the emails of two users, which is useful if someone once used an email
-- address, and then has signed up with a new email and actually wants to use that
-- to access their original account

SELECT s.subdomain_key
      ,p.profile_id
      ,p.profile_name
      ,u.user_id
      ,u.email
      ,u.canonical_email
      ,p.comment_count
      ,p.last_active
      ,CASE WHEN ak.attribute_id IS NULL THEN NULL ELSE 'member' END AS member
  FROM profiles p
  JOIN users u ON p.user_id = u.user_id
  JOIN sites s ON p.site_id = s.site_id
 LEFT JOIN attribute_keys ak ON ak.item_id = p.profile_id AND ak.item_type_id = 3 
 WHERE u.canonical_email IN (
           canonical_email('profile1@gmail.com'),
           canonical_email('user12345@gmail.com')
      );
   --AND p.profile_name IN ('user12345','profile1');

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
