package handler

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type metadataHandlerFake struct {
	repository.ProjectRepo
	project         *domain.Project
	err             error
	statsCalls      int
	individualCalls int
	writes          int
}

func (f *metadataHandlerFake) GetByID(context.Context, uuid.UUID) (*domain.Project, error) {
	return f.project, f.err
}
func (f *metadataHandlerFake) Create(_ context.Context, p *domain.Project) error {
	f.project = p
	f.writes++
	return f.err
}
func (f *metadataHandlerFake) Update(_ context.Context, p *domain.Project) error {
	f.project = p
	f.writes++
	return f.err
}
func (f *metadataHandlerFake) ListByWorkspace(context.Context, uuid.UUID) ([]domain.Project, error) {
	return []domain.Project{*f.project}, f.err
}
func (f *metadataHandlerFake) ListByTeam(context.Context, uuid.UUID) ([]domain.Project, error) {
	return []domain.Project{*f.project}, f.err
}
func (f *metadataHandlerFake) IssueStatsByWorkspace(context.Context, uuid.UUID) (map[uuid.UUID]dto.ProjectProgressResponse, error) {
	f.statsCalls++
	return map[uuid.UUID]dto.ProjectProgressResponse{f.project.ID: {Total: 4, Completed: 2, Cancelled: 1}}, f.err
}
func (f *metadataHandlerFake) IssueStats(context.Context, uuid.UUID) (int, int, int, error) {
	f.individualCalls++
	return 4, 2, 1, f.err
}
func TestProjectMetadataHTTPContract(t *testing.T) {
	ws := uuid.New()
	repo := &metadataHandlerFake{project: &domain.Project{ID: uuid.New(), WorkspaceID: ws, Name: "Original"}}
	h := NewProjectHandler(service.NewProjectService(repo))
	invoke := func(method, body string, workspace uuid.UUID, handler echo.HandlerFunc) *httptest.ResponseRecorder {
		e := echo.New()
		r := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c := e.NewContext(req, r)
		c.SetParamNames("id", "teamId")
		c.SetParamValues(repo.project.ID.String(), uuid.New().String())
		c.Set("workspace", &domain.Workspace{ID: workspace})
		require.NoError(t, handler(c))
		return r
	}
	for _, body := range []string{`{"status":"invalid"}`, `{"team_id":"oops"}`, `{"name":" "}`, `{"start_date":"2026-10-02T00:00:00Z"}`, `{"unknown":true}`, `{} {}`, `null`, `[]`} {
		require.Equal(t, 400, invoke(http.MethodPatch, body, ws, h.Update).Code, body)
	}
	require.Zero(t, repo.writes)
	require.Equal(t, 404, invoke(http.MethodGet, "", uuid.New(), h.Get).Code)
	require.Equal(t, 404, invoke(http.MethodPatch, `{}`, uuid.New(), h.Update).Code)
	r := invoke(http.MethodPatch, `{"start_date":"2026-10-02","target_date":"2026-10-03"}`, ws, h.Update)
	require.Equal(t, 200, r.Code)
	require.Contains(t, r.Body.String(), `"start_date":"2026-10-02T00:00:00Z"`)
	r = invoke(http.MethodPatch, `{"start_date":null,"target_date":null}`, ws, h.Update)
	require.Equal(t, 200, r.Code)
	require.Contains(t, r.Body.String(), `"start_date":null`)
	require.Equal(t, 200, invoke(http.MethodGet, "", ws, h.List).Code)
	require.Equal(t, 1, repo.statsCalls)
	require.Zero(t, repo.individualCalls)
	r = invoke(http.MethodGet, "", uuid.New(), h.ListByTeam)
	require.Equal(t, 200, r.Code)
	require.JSONEq(t, `[]`, r.Body.String())
	repo.err = errors.New("database unavailable")
	require.Equal(t, 500, invoke(http.MethodPatch, `{}`, ws, h.Update).Code)
	require.Equal(t, 500, invoke(http.MethodGet, "", ws, h.Get).Code)
}
