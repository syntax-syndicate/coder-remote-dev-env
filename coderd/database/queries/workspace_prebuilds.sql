-- name: UpsertWorkspacePrebuild :one
INSERT INTO workspace_prebuilds (id, name, replicas, organization_id, template_id, template_version_id, created_by)
VALUES (@id, @name, @replicas, @organization_id, @template_id, @template_version_id, @created_by)
ON CONFLICT (id) DO UPDATE
    SET name                = @name,
        replicas            = @replicas,
        organization_id     = @organization_id,
        template_id         = @template_id,
        template_version_id = @template_version_id,
        created_by          = @created_by
RETURNING *;

-- name: GetWorkspacePrebuilds :many
SELECT * FROM workspace_prebuilds;

-- name: GetWorkspacePrebuildByID :one
SELECT * FROM workspace_prebuilds WHERE id = @id;

-- name: GetWorkspacePrebuildParameters :many
SELECT * FROM workspace_prebuild_parameters WHERE workspace_prebuild_id = $1;