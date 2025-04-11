package coderd

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/codersdk/wsjson"
	"github.com/coder/preview"
	previewtypes "github.com/coder/preview/types"
	"github.com/coder/websocket"
	"golang.org/x/xerrors"
)

// @Summary Open dynamic parameters WebSocket by template version
// @ID open-dynamic-parameters-websocket-by-template-version
// @Security CoderSessionToken
// @Tags Templates Workspaces
// @Param user path string true "Template version ID" format(uuid)
// @Param templateversion path string true "Template version ID" format(uuid)
// @Success 101
// @Router /users/{user}/templateversion/{templateversion}/parameters [get]
func (api *API) templateVersionDynamicParameters(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpmw.UserParam(r)
	templateVersion := httpmw.TemplateVersionParam(r)

	key, err := api.Database.GetGitSSHKey(ctx, user.ID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error fetching SSH key.",
			Detail:  err.Error(),
		})
		return
	}

	groups, err := api.Database.GetGroups(ctx, database.GetGroupsParams{
		OrganizationID: templateVersion.OrganizationID,
		HasMemberID:    user.ID,
	})
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error fetching SSH key.",
			Detail:  err.Error(),
		})
		return
	}
	groupNames := make([]string, 0, len(groups))
	for _, it := range groups {
		groupNames = append(groupNames, it.Group.Name)
	}

	orgRoles, err := api.Database.GetOrganizationMemberRoles(ctx, database.GetOrganizationMemberRolesParams{
		OrganizationID: templateVersion.OrganizationID,
		UserID:         user.ID,
	})
	ownerRoles := make([]previewtypes.WorkspaceOwnerRBACRole, 0, len(user.RBACRoles)+len(orgRoles))
	for _, it := range user.RBACRoles {
		ownerRoles = append(ownerRoles, previewtypes.WorkspaceOwnerRBACRole{
			Name: it,
		})
	}
	for _, it := range orgRoles {
		ownerRoles = append(ownerRoles, previewtypes.WorkspaceOwnerRBACRole{
			Name:  it,
			OrgID: templateVersion.OrganizationID,
		})
	}

	// Check that the job has completed successfully
	job, err := api.Database.GetProvisionerJobByID(ctx, templateVersion.JobID)
	if httpapi.Is404Error(err) {
		httpapi.ResourceNotFound(rw)
		return
	}
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error fetching provisioner job.",
			Detail:  err.Error(),
		})
		return
	}
	if !job.CompletedAt.Valid {
		httpapi.Write(ctx, rw, http.StatusTooEarly, codersdk.Response{
			Message: "Template version job has not finished",
		})
		return
	}

	// Having the Terraform plan available for the evaluation engine is helpful
	// for populating values from data blocks, but isn't strictly required. If
	// we don't have a cached plan available, we just use an empty one instead.
	plan := json.RawMessage("{}")
	tf, err := api.Database.GetTemplateVersionTerraformValues(ctx, templateVersion.ID)
	if err == nil {
		plan = tf.CachedPlan
	} else if !xerrors.Is(err, sql.ErrNoRows) {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to retrieve Terraform values for template version",
			Detail:  err.Error(),
		})
		return
	}

	input := preview.Input{
		PlanJSON:        plan,
		ParameterValues: map[string]string{},
		Owner: previewtypes.WorkspaceOwner{
			ID:           user.ID,
			Name:         user.Username,
			FullName:     user.Name,
			Email:        user.Email,
			LoginType:    string(user.LoginType),
			RBACRoles:    ownerRoles,
			SSHPublicKey: key.PublicKey,
			Groups:       groupNames,
		},
	}

	// nolint:gocritic // We need to fetch the templates files for the Terraform
	// evaluator, and the user likely does not have permission.
	fileCtx := dbauthz.AsProvisionerd(ctx)
	fileID, err := api.Database.GetFileIDByTemplateVersionID(fileCtx, templateVersion.ID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error finding template version Terraform.",
			Detail:  err.Error(),
		})
		return
	}

	fs, err := api.FileCache.Acquire(fileCtx, fileID)
	defer api.FileCache.Release(fileID)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusNotFound, codersdk.Response{
			Message: "Internal error fetching template version Terraform.",
			Detail:  err.Error(),
		})
		return
	}

	conn, err := websocket.Accept(rw, r, nil)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusUpgradeRequired, codersdk.Response{
			Message: "Failed to accept WebSocket.",
			Detail:  err.Error(),
		})
		return
	}

	stream := wsjson.NewStream[codersdk.DynamicParametersRequest, codersdk.DynamicParametersResponse](conn, websocket.MessageText, websocket.MessageText, api.Logger)

	// Send an initial form state, computed without any user input.
	result, diagnostics := preview.Preview(ctx, input, fs)
	response := codersdk.DynamicParametersResponse{
		ID:          -1,
		Diagnostics: previewtypes.Diagnostics(diagnostics),
	}
	if result != nil {
		response.Parameters = result.Parameters
	}
	err = stream.Send(response)
	if err != nil {
		stream.Drop()
		return
	}

	// As the user types into the form, reprocess the state using their input,
	// and respond with updates.
	updates := stream.Chan()
	for {
		select {
		case <-ctx.Done():
			stream.Close(websocket.StatusGoingAway)
			return
		case update, ok := <-updates:
			if !ok {
				// The connection has been closed, so there is no one to write to
				return
			}
			input.ParameterValues = update.Inputs
			result, diagnostics := preview.Preview(ctx, input, fs)
			response := codersdk.DynamicParametersResponse{
				ID:          update.ID,
				Diagnostics: previewtypes.Diagnostics(diagnostics),
			}
			if result != nil {
				response.Parameters = result.Parameters
			}
			err = stream.Send(response)
			if err != nil {
				stream.Drop()
				return
			}
		}
	}
}
