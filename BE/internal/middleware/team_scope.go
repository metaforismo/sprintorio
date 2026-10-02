package middleware

import (
	"context"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/pkg/response"
	"net/http"
)

type teamScopeReader interface {
	GetByID(context.Context, uuid.UUID) (*domain.Team, error)
}
type statusScopeReader interface {
	GetByID(context.Context, uuid.UUID) (*domain.TeamStatus, error)
}
type cycleScopeReader interface {
	GetByID(context.Context, uuid.UUID) (*domain.Cycle, error)
}

// TeamResourceScope applies to every nested team route, including statistics
// and mutations. Child IDs must belong to the team named in the URL.
func TeamResourceScope(teams teamScopeReader, statuses statusScopeReader, cycles cycleScopeReader) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Param("teamId") == "" {
				return next(c)
			}
			teamID, err := uuid.Parse(c.Param("teamId"))
			if err != nil {
				return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid team ID")
			}
			ctx := c.Request().Context()
			team, err := teams.GetByID(ctx, teamID)
			if err != nil {
				return response.InternalError(c)
			}
			ws := GetWorkspace(c)
			if ws == nil || team == nil || team.WorkspaceID != ws.ID {
				return response.NotFound(c, "Team")
			}
			if raw := c.Param("statusId"); raw != "" {
				id, err := uuid.Parse(raw)
				if err != nil {
					return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid status ID")
				}
				status, err := statuses.GetByID(ctx, id)
				if err != nil {
					return response.InternalError(c)
				}
				if status == nil || status.TeamID != teamID {
					return response.NotFound(c, "Status")
				}
			}
			if raw := c.Param("cycleId"); raw != "" {
				id, err := uuid.Parse(raw)
				if err != nil {
					return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid cycle ID")
				}
				cycle, err := cycles.GetByID(ctx, id)
				if err != nil {
					return response.InternalError(c)
				}
				if cycle == nil || cycle.TeamID != teamID {
					return response.NotFound(c, "Cycle")
				}
			}
			return next(c)
		}
	}
}
