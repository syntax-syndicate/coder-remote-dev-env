DROP TABLE IF EXISTS workspace_prebuilds CASCADE;
ALTER TABLE workspaces DROP COLUMN IF EXISTS prebuild_id;
ALTER TABLE workspaces DROP COLUMN IF EXISTS prebuild_assigned;