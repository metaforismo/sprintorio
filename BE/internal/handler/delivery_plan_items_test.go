package handler

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeliveryPlanItemsHTTPContract(t *testing.T) {
	ws, id := uuid.New(), uuid.New()
	repo := &planHandlerFake{project: &domain.Project{ID: id, WorkspaceID: ws}}
	h := NewProjectHandler(service.NewProjectService(repo))
	invoke := func(method, body string, workspace uuid.UUID) *httptest.ResponseRecorder {
		e := echo.New()
		r := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c := e.NewContext(req, r)
		c.SetParamNames("id")
		c.SetParamValues(id.String())
		c.Set("workspace", &domain.Workspace{ID: workspace})
		var err error
		if method == http.MethodGet {
			err = h.GetDeliveryPlanSummary(c)
		} else {
			err = h.UpdateDeliveryPlanItems(c)
		}
		require.NoError(t, err)
		return r
	}
	for _, body := range []string{`{}`, `{"version":-1}`, `{"version":0,"unknown":1}`, `{"version":0} {}`, `{"version":0}`, `{"version":0,"test_cases":{"remove":["missing"]}}`, `{"version":0,"test_cases":{"upsert":[{"id":"t","title":"Test","status":"passed"}]}}`} {
		require.Equal(t, 400, invoke(http.MethodPatch, body, ws).Code, body)
	}
	require.Equal(t, 404, invoke(http.MethodGet, "", uuid.New()).Code)
	require.Equal(t, 404, invoke(http.MethodPatch, `{"version":0,"test_cases":{"upsert":[{"id":"t","title":"Test","status":"not_run"}]}}`, uuid.New()).Code)
	require.Equal(t, 200, invoke(http.MethodPatch, `{"version":0,"milestones":null,"test_cases":{"upsert":[{"id":"t","title":"Test","status":"passed","evidence":"Actual run"}]}}`, ws).Code)
	require.Equal(t, 409, invoke(http.MethodPatch, `{"version":0,"test_cases":{"remove":["t"]}}`, ws).Code)
	result := invoke(http.MethodGet, "", ws)
	require.Contains(t, result.Body.String(), `"ready":false`)
	require.Contains(t, result.Body.String(), `"basis":"saved_plan_and_test_evidence"`)
	require.Contains(t, result.Body.String(), `"version":1`)
}
