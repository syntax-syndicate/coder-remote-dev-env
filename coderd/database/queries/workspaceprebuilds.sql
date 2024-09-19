-- name: UpsertWorkspacePrebuildPool :one
INSERT INTO workspace_prebuild_pool (id, name, count, organization_id, template_id, template_version_id, created_by)
VALUES (@id, @name, @count, @organization_id, @template_id, @template_version_id, @created_by)
ON CONFLICT (id) DO UPDATE
    SET name                = @name,
        count               = @count,
        organization_id     = @organization_id,
        template_id         = @template_id,
        template_version_id = @template_version_id,
        created_by          = @created_by
RETURNING *;

-- TODO: rename these to have Pool suffix

-- name: GetWorkspacePrebuilds :many
SELECT *
FROM workspace_prebuild_pool;

-- name: GetWorkspacePrebuildByID :one
SELECT *
FROM workspace_prebuild_pool
WHERE id = @id;

--
-- TODO: create view for latest build!! WE NEED TO SEE INTERMEDIARY STATUSES, NOT JUST TERMINAL ONES.
--          i.e. when we list the unassigned prebuilds we must exclude the ones which are in the process of transferring
--

-- name: GetPrebuildsByPoolID :many
SELECT w.*
FROM workspace_prebuild_pool wpp
         INNER JOIN workspaces w ON wpp.id = w.prebuild_id
         INNER JOIN LATERAL (
    SELECT wb.transition
    FROM workspace_builds wb
             LEFT JOIN provisioner_jobs pj ON pj.id = wb.job_id
    WHERE wb.workspace_id = w.id
      -- we only consider workspaces which are fully built
      AND pj.completed_at IS NOT NULL
      AND pj.canceled_at IS NULL
      AND pj.error IS NULL
    ORDER BY build_number DESC
    LIMIT 1
    ) latest_build ON TRUE
-- we only consider workspaces which are not deleted, unassigned, and in any state
WHERE w.deleted = false
  AND w.prebuild_id = @prebuild_id::uuid
  AND w.prebuild_assigned = false
GROUP BY latest_build.transition, w.id;

-- name: GetUnassignedPrebuildsByPoolID :many
SELECT w.*
FROM workspace_prebuild_pool wpp
         INNER JOIN workspaces w ON wpp.id = w.prebuild_id
         INNER JOIN LATERAL (
    SELECT wb.transition
    FROM workspace_builds wb
             LEFT JOIN provisioner_jobs pj ON pj.id = wb.job_id
    WHERE wb.workspace_id = w.id
      -- we only consider workspaces which are fully built
      AND pj.completed_at IS NOT NULL
      AND pj.canceled_at IS NULL
      AND pj.error IS NULL
    ORDER BY build_number DESC
    LIMIT 1
    ) latest_build ON TRUE
-- we only consider workspaces which are not deleted, unassigned, and in a "stop" state
WHERE w.deleted = false
  AND w.prebuild_id = @prebuild_id::uuid
  AND w.prebuild_assigned = false
  AND latest_build.transition = 'stop'::workspace_transition
GROUP BY latest_build.transition, w.id;

-- name: GetMatchingPrebuilds :many
SELECT *
FROM workspace_prebuild_pool
WHERE template_version_id = @template_version_id;

-- name: MarkWorkspacePrebuildAssigned :exec
UPDATE workspaces
SET prebuild_assigned = true
WHERE id = $1;

-- SELECT wp.id                                                        AS prebuild_id,
--        latest_build.template_id,
--        latest_build.template_version_id,
--        md5(string_agg(wpp.name || wpp.value, '' ORDER BY wpp.name)) AS params_hash
-- FROM workspace_prebuild_pool wp
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