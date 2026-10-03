package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/pkg/response"
)

type workspaceResourceReader interface {
	Accessible(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID, bool) (bool, error)
}

// WorkspaceResourceScope binds UUID product and admin resources to the route's
// workspace for browser sessions and agent credentials alike.
func WorkspaceResourceScope(resources workspaceResourceReader) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Param("id") == "" {
				return next(c)
			}
			path := strings.TrimPrefix(c.Path(), "/api/workspaces/:slug/")
			resource := strings.Split(path, "/")[0]
			if strings.HasPrefix(path, "github/repos/:id") {
				resource = "github/repos"
			}
			switch resource {
			case "views", "issue-templates", "shared-links", "webhooks", "labels", "projects", "favorites", "github/repos":
			default:
				return next(c)
			}
			id, err := uuid.Parse(c.Param("id"))
			if err != nil {
				return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid resource ID")
			}
			ws := GetWorkspace(c)
			if ws == nil {
				return response.Forbidden(c)
			}
			allowed, err := resources.Accessible(c.Request().Context(), resource, id, ws.ID, GetUserID(c), c.Request().Method != http.MethodGet)
			if err != nil {
				return response.InternalError(c)
			}
			if !allowed {
				return response.NotFound(c, "Resource")
			}
			return next(c)
		}
	}
}
