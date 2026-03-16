-- Remove single manager constraint
DROP INDEX IF EXISTS idx_one_manager_per_building;

-- Note: PostgreSQL doesn't support removing values from enums easily.
-- The pending_approval value will remain but won't be used.
