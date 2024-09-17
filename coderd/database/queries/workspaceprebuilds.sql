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
SELECT *
FROM workspace_prebuilds;

-- name: GetWorkspacePrebuildByID :one
SELECT *
FROM workspace_prebuilds
WHERE id = @id;

-- name: GetWorkspacesByPrebuildID :many
SELECT *
FROM workspaces
WHERE prebuild_id = @id::uuid;

-- name: GetMatchingPrebuilds :many
SELECT *
FROM workspace_prebuilds
WHERE template_id = @template_id AND template_version_id = @template_version_id;

-- SELECT wp.id                                                        AS prebuild_id,
--        latest_build.template_id,
--        latest_build.template_version_id,
--        md5(string_agg(wpp.name || wpp.value, '' ORDER BY wpp.name)) AS params_hash
-- FROM workspace_prebuilds wp
--          LEFT JOIN workspace_prebuild_parameters wpp ON wpp.workspace_prebuild_id = wp.id
--          LEFT JOIN (SELECT w.template_id,
--                            w.prebuild_id,
--                            wb.template_version_id,
--                            pj.id,
--                            pj.job_status,
--                            wb.transition,
--                            wb.build_number
--                     FROM workspace_builds wb
--                              INNER JOIN workspaces w ON w.prebuild_id = wp.id
--                              LEFT JOIN provisioner_jobs pj ON pj.id = wb.job_id
--                     WHERE wb.workspace_id = @workspace_id
--                     ORDER BY wb.build_number DESC
--                     LIMIT 1) latest_build ON latest_build.prebuild_id = wp.id
-- WHERE latest_build.template_id = @template_id
-- GROUP BY wp.id, latest_build.template_id, latest_build.template_version_id;