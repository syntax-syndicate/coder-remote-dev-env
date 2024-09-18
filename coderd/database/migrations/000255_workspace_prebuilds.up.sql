CREATE TABLE workspace_prebuild_pool
(
    id                  uuid        NOT NULL,
    name                text        not null,
    count               int         NOT NULL,
    organization_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    template_id         uuid        NOT NULL REFERENCES templates (id) ON DELETE CASCADE,
    template_version_id uuid        NOT NULL REFERENCES template_versions (id) ON DELETE CASCADE,
    parameters          jsonb       NOT NULL DEFAULT '[]'::jsonb,
    ignored_parameters  text[]      NOT NULL DEFAULT array[]::text[],
    created_by          uuid        NULL REFERENCES users (id) ON DELETE SET NULL,
    -- TODO: autostart schedule?
    created_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (name),
    UNIQUE (template_id, template_version_id, parameters)
);

ALTER TABLE workspaces
    ADD COLUMN IF NOT EXISTS prebuild_id       uuid NULL REFERENCES workspace_prebuild_pool (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS prebuild_assigned bool NULL; -- this field is nullable because it only makes sense if prebuild_id is set