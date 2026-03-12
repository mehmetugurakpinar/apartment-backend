-- User follows table
CREATE TABLE IF NOT EXISTS user_follows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(follower_id, following_id),
    CHECK (follower_id != following_id)
);

CREATE INDEX IF NOT EXISTS idx_user_follows_follower ON user_follows(follower_id);
CREATE INDEX IF NOT EXISTS idx_user_follows_following ON user_follows(following_id);

-- Add repost columns to timeline_posts
ALTER TABLE timeline_posts ADD COLUMN IF NOT EXISTS repost_count INTEGER DEFAULT 0;
ALTER TABLE timeline_posts ADD COLUMN IF NOT EXISTS original_post_id UUID REFERENCES timeline_posts(id) ON DELETE SET NULL;
ALTER TABLE timeline_posts ADD COLUMN IF NOT EXISTS is_repost BOOLEAN DEFAULT false;

-- Add follower/following counts to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS follower_count INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS following_count INTEGER DEFAULT 0;
