ALTER TABLE users DROP COLUMN IF EXISTS following_count;
ALTER TABLE users DROP COLUMN IF EXISTS follower_count;

ALTER TABLE timeline_posts DROP COLUMN IF EXISTS is_repost;
ALTER TABLE timeline_posts DROP COLUMN IF EXISTS original_post_id;
ALTER TABLE timeline_posts DROP COLUMN IF EXISTS repost_count;

DROP TABLE IF EXISTS user_follows;
