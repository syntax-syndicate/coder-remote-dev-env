DROP TABLE IF EXISTS workspace_prebuild_pool CASCADE;
ALTER TABLE workspaces DROP COLUMN IF EXISTS prebuild_id;
ALTER TABLE workspaces DROP COLUMN IF EXISTS prebuild_assigned;