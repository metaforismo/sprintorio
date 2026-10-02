package handler

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/middleware"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type planHandlerFake struct {
	repository.ProjectRepo
	project *domain.Project
}

func (f *planHandlerFake) GetByID(context.Context, uuid.UUID) (*domain.Project, error) {
	return f.project, nil
}
func (f *planHandlerFake) UpdateDeliveryPlan(_ context.Context, _ uuid.UUID, _ uuid.UUID, raw json.RawMessage, version int) (bool, error) {
	f.project.DeliveryPlan = raw
	f.project.DeliveryPlanVersion++
	return true, nil
}
func TestDeliveryPlanHTTPContract(t *testing.T) {
	ws := uuid.New()
	id := uuid.New()
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
			err = h.GetDeliveryPlan(c)
		} else {
			err = h.UpdateDeliveryPlan(c)
		}
		require.NoError(t, err)
		return r
	}
	r := invoke(http.MethodGet, "", ws)
	require.Equal(t, 200, r.Code)
	require.JSONEq(t, `{"plan":{"product_name":"","objective":"","success_metric":"","target_release":"","milestones":[],"test_cases":[]},"version":0}`, r.Body.String())
	require.Equal(t, 404, invoke(http.MethodGet, "", uuid.New()).Code)
	for _, body := range []string{`{}`, `{"plan":{},"version":-1}`, `{"plan":{},"version":0,"unknown":1}`, `{"plan":{},"version":0} {}`, `{"plan":{"test_cases":[{"id":"1","title":"Check","status":"passed","evidence":""}]},"version":0}`} {
		require.Equal(t, 400, invoke(http.MethodPatch, body, ws).Code, body)
	}
	require.Equal(t, 404, invoke(http.MethodPatch, `{"plan":{},"version":0}`, uuid.New()).Code)
	require.Equal(t, 200, invoke(http.MethodPatch, `{"plan":{"objective":"Saved"},"version":0}`, ws).Code)
	r = invoke(http.MethodPatch, `{"plan":{},"version":0}`, ws)
	require.Equal(t, 409, r.Code)
	require.Contains(t, r.Body.String(), "DELIVERY_PLAN_CONFLICT")
	r = invoke(http.MethodGet, "", ws)
	require.Contains(t, r.Body.String(), `"objective":"Saved"`)
	require.Contains(t, r.Body.String(), `"version":1`)
}

func TestDeliveryPlanMutationRequiresProjectManage(t *testing.T) {
	for _, role := range []string{domain.RoleGuest, "", "unknown"} {
		e := echo.New()
		recorder := httptest.NewRecorder()
		c := e.NewContext(httptest.NewRequest(http.MethodPatch, "/", nil), recorder)
		c.Set("workspace_role", role)
		called := false
		err := middleware.RequirePermission("project:manage")(func(echo.Context) error { called = true; return nil })(c)
		require.NoError(t, err)
		require.False(t, called)
		require.Equal(t, http.StatusForbidden, recorder.Code)
	}
}
