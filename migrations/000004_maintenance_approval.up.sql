-- Add pending_approval status to maintenance_status enum
ALTER TYPE maintenance_status ADD VALUE IF NOT EXISTS 'pending_approval' BEFORE 'open';

-- Add unique partial index to enforce single manager per building
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_manager_per_building
    ON building_members (building_id)
    WHERE role = 'building_manager';
