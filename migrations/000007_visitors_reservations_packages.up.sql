-- Enable btree_gist for EXCLUDE constraints
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Visitor Management
CREATE TABLE visitor_passes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    unit_id UUID REFERENCES units(id) ON DELETE SET NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    visitor_name VARCHAR(255) NOT NULL,
    visitor_phone VARCHAR(50),
    visitor_plate VARCHAR(50),
    purpose VARCHAR(255),
    expected_at TIMESTAMPTZ,
    checked_in_at TIMESTAMPTZ,
    checked_out_at TIMESTAMPTZ,
    qr_code VARCHAR(255) UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, checked_in, checked_out, cancelled
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_visitor_passes_building ON visitor_passes(building_id);
CREATE INDEX idx_visitor_passes_status ON visitor_passes(building_id, status);
CREATE INDEX idx_visitor_passes_qr ON visitor_passes(qr_code);

-- Common Area Reservations
CREATE TABLE common_areas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    capacity INT DEFAULT 0,
    rules TEXT,
    open_time TIME DEFAULT '08:00',
    close_time TIME DEFAULT '22:00',
    requires_approval BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_common_areas_building ON common_areas(building_id);

CREATE TABLE reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    common_area_id UUID NOT NULL REFERENCES common_areas(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    title VARCHAR(255),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    guest_count INT DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, cancelled
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT no_overlap EXCLUDE USING gist (
        common_area_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    ) WHERE (status IN ('pending', 'approved'))
);

CREATE INDEX idx_reservations_area ON reservations(common_area_id);
CREATE INDEX idx_reservations_user ON reservations(user_id);
CREATE INDEX idx_reservations_building ON reservations(building_id);
CREATE INDEX idx_reservations_time ON reservations(common_area_id, start_time, end_time);

-- Package / Cargo Tracking
CREATE TABLE packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    unit_id UUID REFERENCES units(id) ON DELETE SET NULL,
    recipient_id UUID REFERENCES users(id) ON DELETE SET NULL,
    carrier VARCHAR(255),
    tracking_number VARCHAR(255),
    description TEXT,
    received_by UUID REFERENCES users(id) ON DELETE SET NULL, -- security/doorman who received
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    picked_up_by UUID REFERENCES users(id) ON DELETE SET NULL,
    picked_up_at TIMESTAMPTZ,
    photo_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'waiting', -- waiting, notified, picked_up
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_packages_building ON packages(building_id);
CREATE INDEX idx_packages_recipient ON packages(recipient_id);
CREATE INDEX idx_packages_status ON packages(building_id, status);
