-- Will swap the emails of two users, which is useful if someone once used an email
-- address, and then has signed up with a new email and actually wants to use that
-- to access their original account

SELECT p.profile_id
      ,p.profile_name
      ,u.user_id
      ,u.email
  FROM profiles p
      ,users u
 WHERE p.user_id = u.user_id
   AND p.site_id = 234
   AND p.profile_name IN ('profile1','user12345');

UPDATE users
   SET email = 'profile1@email'
 WHERE user_id = 44194; -- user ID of user12345

UPDATE users
   SET email = 'user12345@email'
 WHERE user_id = 3619; -- user ID of profile1
