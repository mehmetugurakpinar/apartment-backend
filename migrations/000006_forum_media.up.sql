-- Forum media/image support
CREATE TABLE IF NOT EXISTS forum_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'image',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_forum_media_post ON forum_media(post_id);
