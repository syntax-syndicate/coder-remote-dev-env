package httpmw

import (
	"context"
	"net/http"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/codersdk"
)

type workspacePrebuildParamContextKey struct{}

func WorkspacePrebuildParam(r *http.Request) database.WorkspacePrebuild {
	workspace, ok := r.Context().Value(workspacePrebuildParamContextKey{}).(database.WorkspacePrebuild)
	if !ok {
		panic("developer error: workspace prebuild param middleware not provided")
	}
	return workspace
}

func ExtractWorkspacePrebuildParam(db database.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			prebuildID, parsed := ParseUUIDParam(rw, r, "prebuildname")
			if !parsed {
				return
			}
			workspace, err := db.GetWorkspacePrebuildByID(ctx, prebuildID)
			if httpapi.Is404Error(err) {
				httpapi.ResourceNotFound(rw)
				return
			}
			if err != nil {
				httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
					Message: "Internal error fetching workspace prebuild.",
					Detail:  err.Error(),
				})
				return
			}

			ctx = context.WithValue(ctx, workspacePrebuildParamContextKey{}, workspace)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
