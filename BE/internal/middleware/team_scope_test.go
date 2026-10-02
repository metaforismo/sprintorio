package middleware

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

type scopeTeams struct {
	team *domain.Team
	err  error
}

func (f scopeTeams) GetByID(context.Context, uuid.UUID) (*domain.Team, error) { return f.team, f.err }

type scopeStatuses struct{ status *domain.TeamStatus }

func (f scopeStatuses) GetByID(context.Context, uuid.UUID) (*domain.TeamStatus, error) {
	return f.status, nil
}

type scopeCycles struct{ cycle *domain.Cycle }

func (f scopeCycles) GetByID(context.Context, uuid.UUID) (*domain.Cycle, error) { return f.cycle, nil }
func TestTeamResourceScopeRejectsForeignParentsAndChildren(t *testing.T) {
	ws, teamID, other := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name   string
		team   *domain.Team
		status *domain.TeamStatus
		cycle  *domain.Cycle
		child  string
		err    error
		want   int
	}{
		{name: "valid team", team: &domain.Team{ID: teamID, WorkspaceID: ws}, want: 204},
		{name: "other workspace", team: &domain.Team{ID: teamID, WorkspaceID: other}, want: 404},
		{name: "missing team", want: 404},
		{name: "foreign status", team: &domain.Team{ID: teamID, WorkspaceID: ws}, status: &domain.TeamStatus{TeamID: other}, child: "statusId", want: 404},
		{name: "valid status", team: &domain.Team{ID: teamID, WorkspaceID: ws}, status: &domain.TeamStatus{TeamID: teamID}, child: "statusId", want: 204},
		{name: "foreign cycle", team: &domain.Team{ID: teamID, WorkspaceID: ws}, cycle: &domain.Cycle{TeamID: other}, child: "cycleId", want: 404},
		{name: "valid cycle", team: &domain.Team{ID: teamID, WorkspaceID: ws}, cycle: &domain.Cycle{TeamID: teamID}, child: "cycleId", want: 204},
		{name: "datastore error", err: errors.New("unavailable"), want: 500},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
				e := echo.New()
				r := httptest.NewRecorder()
				c := e.NewContext(httptest.NewRequest(method, "/", nil), r)
				names := []string{"teamId"}
				values := []string{teamID.String()}
				if test.child != "" {
					names = append(names, test.child)
					values = append(values, uuid.New().String())
				}
				c.SetParamNames(names...)
				c.SetParamValues(values...)
				c.Set("workspace", &domain.Workspace{ID: ws})
				called := false
				err := TeamResourceScope(scopeTeams{test.team, test.err}, scopeStatuses{test.status}, scopeCycles{test.cycle})(func(c echo.Context) error { called = true; return c.NoContent(204) })(c)
				require.NoError(t, err)
				require.Equal(t, test.want, r.Code)
				require.Equal(t, test.want == 204, called)
			}
		})
	}
}
