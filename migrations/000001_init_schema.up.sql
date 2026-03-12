-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enum types
CREATE TYPE user_role AS ENUM ('super_admin', 'building_manager', 'resident', 'security');
CREATE TYPE unit_status AS ENUM ('occupied', 'vacant', 'maintenance');
CREATE TYPE resident_type AS ENUM ('owner', 'tenant');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'late');
CREATE TYPE maintenance_priority AS ENUM ('emergency', 'high', 'normal', 'low');
CREATE TYPE maintenance_status AS ENUM ('open', 'in_progress', 'resolved', 'closed');
CREATE TYPE timeline_post_type AS ENUM ('text', 'photo', 'poll', 'event', 'alert');
CREATE TYPE visibility_type AS ENUM ('building', 'neighborhood', 'public');

-- ============================================
-- USERS & AUTH
-- ============================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    full_name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    role user_role NOT NULL DEFAULT 'resident',
    fcm_token TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- ============================================
-- BUILDINGS & UNITS
-- ============================================
CREATE TABLE buildings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    total_units INTEGER NOT NULL DEFAULT 0,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    image_url TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE units (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    block VARCHAR(10),
    floor INTEGER NOT NULL,
    unit_number VARCHAR(20) NOT NULL,
    area_sqm DECIMAL(8,2),
    owner_id UUID REFERENCES users(id),
    status unit_status DEFAULT 'vacant',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(building_id, block, unit_number)
);

CREATE INDEX idx_units_building_id ON units(building_id);

CREATE TABLE unit_residents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type resident_type NOT NULL DEFAULT 'tenant',
    move_in_date DATE NOT NULL DEFAULT CURRENT_DATE,
    move_out_date DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_unit_residents_unit_id ON unit_residents(unit_id);
CREATE INDEX idx_unit_residents_user_id ON unit_residents(user_id);

CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    plate VARCHAR(20) NOT NULL,
    type VARCHAR(50),
    parking_spot VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- FINANCIAL
-- ============================================
CREATE TABLE dues_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    period_month INTEGER NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    period_year INTEGER NOT NULL,
    due_date DATE NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dues_plans_building_id ON dues_plans(building_id);

CREATE TABLE due_payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    dues_plan_id UUID NOT NULL REFERENCES dues_plans(id) ON DELETE CASCADE,
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    paid_amount DECIMAL(12,2),
    paid_at TIMESTAMPTZ,
    receipt_url TEXT,
    status payment_status DEFAULT 'pending',
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(dues_plan_id, unit_id)
);

CREATE INDEX idx_due_payments_unit_id ON due_payments(unit_id);
CREATE INDEX idx_due_payments_status ON due_payments(status);

CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    description TEXT,
    date DATE NOT NULL,
    receipt_url TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_expenses_building_id ON expenses(building_id);

-- ============================================
-- MAINTENANCE
-- ============================================
CREATE TABLE maintenance_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    unit_id UUID REFERENCES units(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    priority maintenance_priority DEFAULT 'normal',
    status maintenance_status DEFAULT 'open',
    assigned_to UUID REFERENCES users(id),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_maintenance_building_id ON maintenance_requests(building_id);
CREATE INDEX idx_maintenance_status ON maintenance_requests(status);

CREATE TABLE maintenance_photos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id UUID NOT NULL REFERENCES maintenance_requests(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE vendors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    phone VARCHAR(20),
    email VARCHAR(255),
    rating DECIMAL(2,1) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- NOTIFICATIONS
-- ============================================
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    building_id UUID REFERENCES buildings(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    data JSONB,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_read_at ON notifications(read_at);

CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL,
    push_enabled BOOLEAN DEFAULT true,
    email_enabled BOOLEAN DEFAULT false,
    UNIQUE(user_id, category)
);

-- ============================================
-- FORUM
-- ============================================
CREATE TABLE forum_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(building_id, slug)
);

CREATE TABLE forum_posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES forum_categories(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    is_pinned BOOLEAN DEFAULT false,
    is_resolved BOOLEAN DEFAULT false,
    upvotes INTEGER DEFAULT 0,
    downvotes INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_forum_posts_building_id ON forum_posts(building_id);
CREATE INDEX idx_forum_posts_category_id ON forum_posts(category_id);

CREATE TABLE forum_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES forum_comments(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_forum_comments_post_id ON forum_comments(post_id);

CREATE TABLE forum_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    value SMALLINT NOT NULL CHECK (value IN (-1, 1)),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(post_id, user_id)
);

-- ============================================
-- TIMELINE (COMMUNITY FEED)
-- ============================================
CREATE TABLE timeline_posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id UUID NOT NULL REFERENCES users(id),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    content TEXT,
    type timeline_post_type DEFAULT 'text',
    visibility visibility_type DEFAULT 'building',
    location_lat DOUBLE PRECISION,
    location_lng DOUBLE PRECISION,
    like_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_timeline_posts_building_id ON timeline_posts(building_id);
CREATE INDEX idx_timeline_posts_created_at ON timeline_posts(created_at DESC);

CREATE TABLE timeline_media (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES timeline_posts(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    type VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE timeline_likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES timeline_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(post_id, user_id)
);

CREATE TABLE timeline_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES timeline_posts(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES timeline_comments(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_timeline_comments_post_id ON timeline_comments(post_id);

CREATE TABLE polls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES timeline_posts(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    ends_at TIMESTAMPTZ,
    total_votes INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE poll_options (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    text VARCHAR(255) NOT NULL,
    vote_count INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0
);

CREATE TABLE poll_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(poll_id, user_id)
);

-- ============================================
-- BUILDING-USER MEMBERSHIP (for access control)
-- ============================================
CREATE TABLE building_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role user_role NOT NULL DEFAULT 'resident',
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(building_id, user_id)
);

CREATE INDEX idx_building_members_building_id ON building_members(building_id);
CREATE INDEX idx_building_members_user_id ON building_members(user_id);

-- ============================================
-- UPDATED_AT TRIGGER
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_buildings_updated_at BEFORE UPDATE ON buildings FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_units_updated_at BEFORE UPDATE ON units FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_due_payments_updated_at BEFORE UPDATE ON due_payments FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_maintenance_updated_at BEFORE UPDATE ON maintenance_requests FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_forum_posts_updated_at BEFORE UPDATE ON forum_posts FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_timeline_posts_updated_at BEFORE UPDATE ON timeline_posts FOR EACH ROW EXECUTE FUNCTION update_updated_at();
