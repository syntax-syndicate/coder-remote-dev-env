package coderd

import (
	"net/http"

	"github.com/coder/coder/v2/coderd/database/db2sdk"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/codersdk"
)

func (api *API) postWorkspacePrebuilds(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	apiKey := httpmw.APIKey(r)
	org := httpmw.OrganizationParam(r)

	// prebuild := httpmw.WorkspacePrebuildParam(r)
	var createPrebuild codersdk.CreateWorkspacePrebuildRequest
	if !httpapi.Read(ctx, rw, r, &createPrebuild) {
		return
	}

	createPrebuild.CreatedBy = apiKey.UserID
	createPrebuild.OrganizationID = org.ID

	pb, err := api.PrebuildsController.CreateNewWorkspacePrebuild(ctx, createPrebuild)
	if err != nil || pb == nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error creating workspace prebuild.",
			Detail:  err.Error(),
		})
		return
	}

	apiPb := db2sdk.WorkspacePrebuild(*pb)
	httpapi.Write(ctx, rw, http.StatusCreated, apiPb)
}
