-- has the effect of moving all comments from one conversation to another
-- conversation and deleting the conversation with no comments.

-- conversation 262166 loses it's comments and it deleted
-- conversation 262165 gets all the comments

UPDATE conversations
   SET is_deleted = TRUE
 WHERE conversation_id = 262166;

UPDATE comments
   SET item_type_id = 6
      ,item_id = 262165
 WHERE item_type_id = 6
   AND item_id = 262166;

UPDATE watchers
   SET item_type_id = 6
      ,item_id = 262165
 WHERE item_type_id = 6
   AND item_id = 262166;

UPDATE updates
   SET item_type_id = 6
      ,item_id = 262165
 WHERE item_type_id = 6
   AND item_id = 262166;
