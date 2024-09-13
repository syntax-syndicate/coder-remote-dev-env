package codersdk

import (
	"time"

	"github.com/google/uuid"
)

type CreateWorkspacePrebuildRequest struct {
	CreatedBy           uuid.UUID                    // Only settable by the API handler.
	TemplateID          uuid.UUID                    `json:"template_id" validate:"required" format:"uuid"`
	TemplateVersionID   uuid.UUID                    `json:"template_version_id" validate:"required" format:"uuid"`
	OrganizationID      uuid.UUID                    // Only settable by the API handler.
	Name                string                       `json:"name" validate:"workspace_prebuild_name,required"`
	RichParameterValues []WorkspacePrebuildParameter `json:"rich_parameter_values,omitempty"`
	Replicas            int                          `json:"replicas" validate:"required"`
}

type WorkspacePrebuild struct {
	ID                uuid.UUID     `json:"id" format:"uuid"`
	Name              string        `json:"name"`
	Replicas          int           `json:"replicas"`
	OrganizationID    uuid.UUID     `json:"organization_id" format:"uuid"`
	TemplateID        uuid.UUID     `json:"template_id" format:"uuid"`
	TemplateVersionID uuid.UUID     `json:"template_version_id" format:"uuid"`
	CreatedBy         uuid.NullUUID `json:"created_by" format:"uuid"`
	CreatedAt         time.Time     `json:"created_at" format:"date-time"`
	UpdatedAt         time.Time     `json:"updated_at" format:"date-time"`
}

type WorkspacePrebuildParameter struct {
	WorkspacePrebuildID uuid.UUID `json:"workspace_prebuild_id" format:"uuid"`
	Name                string    `json:"name"`
	Value               string    `json:"value"`
}
