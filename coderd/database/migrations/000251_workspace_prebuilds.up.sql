CREATE TABLE workspace_prebuilds
(
    id                  uuid        NOT NULL,
    name                text        not null,
    replicas            int         NOT NULL,
    organization_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    template_id         uuid        NOT NULL REFERENCES templates (id) ON DELETE CASCADE,
    template_version_id uuid        NOT NULL REFERENCES template_versions (id) ON DELETE CASCADE,
    parameters          jsonb       NOT NULL DEFAULT '[]'::jsonb,
    created_by          uuid        NULL REFERENCES users (id) ON DELETE SET NULL,
    -- TODO: autostart schedule?
    created_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (name)
);

ALTER TABLE workspaces
    ADD COLUMN IF NOT EXISTS prebuild_id uuid NULL REFERENCES workspace_prebuilds (id) ON DELETE SET NULL;