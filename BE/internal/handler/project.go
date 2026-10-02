package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	"github.com/metaforismo/sprintorio/BE/pkg/response"
	"github.com/metaforismo/sprintorio/BE/pkg/validate"
)

type ProjectHandler struct {
	projectSvc *service.ProjectService
}

func NewProjectHandler(projectSvc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectSvc: projectSvc}
}

func (h *ProjectHandler) List(c echo.Context) error {
	ws := c.Get("workspace").(*domain.Workspace)
	projects, err := h.projectSvc.ListByWorkspace(c.Request().Context(), ws.ID)
	if err != nil {
		return response.InternalError(c)
	}
	statsByID, err := h.projectSvc.GetWorkspaceStats(c.Request().Context(), ws.ID)
	if err != nil {
		return response.InternalError(c)
	}
	resp := make([]dto.ProjectResponse, len(projects))
	for i, p := range projects {
		r := toProjectResponse(p)
		stats := statsByID[p.ID]
		r.Progress = &stats
		resp[i] = r
	}
	return response.Success(c, http.StatusOK, resp)
}

func (h *ProjectHandler) Create(c echo.Context) error {
	var req dto.CreateProjectRequest
	if err := bindProjectRequest(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
	}
	if err := validate.Struct(&req); err != nil {
		details := make([]dto.ErrorDetail, 0)
		for _, e := range validate.FormatErrors(err) {
			details = append(details, dto.ErrorDetail{Field: e["field"], Message: e["message"]})
		}
		return response.ValidationError(c, details)
	}

	ws := c.Get("workspace").(*domain.Workspace)
	project, err := h.projectSvc.Create(c.Request().Context(), ws.ID, req)
	if err != nil {
		return projectError(c, err)
	}
	return response.Success(c, http.StatusCreated, toProjectResponse(*project))
}

func (h *ProjectHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid project ID")
	}
	project, err := h.projectSvc.GetByID(c.Request().Context(), id)
	if err != nil {
		return response.InternalError(c)
	}
	ws := c.Get("workspace").(*domain.Workspace)
	if project == nil || project.WorkspaceID != ws.ID {
		return response.NotFound(c, "Project")
	}
	r := toProjectResponse(*project)
	stats, err := h.projectSvc.GetStats(c.Request().Context(), project.ID)
	if err != nil {
		return response.InternalError(c)
	}
	if stats != nil {
		r.Progress = stats
	}
	return response.Success(c, http.StatusOK, r)
}

func (h *ProjectHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid project ID")
	}
	var req dto.UpdateProjectRequest
	if err := bindProjectRequest(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
	}
	ws := c.Get("workspace").(*domain.Workspace)
	project, err := h.projectSvc.Update(c.Request().Context(), ws.ID, id, req)
	if err != nil {
		return projectError(c, err)
	}
	return response.Success(c, http.StatusOK, toProjectResponse(*project))
}

func (h *ProjectHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid project ID")
	}
	ws := c.Get("workspace").(*domain.Workspace)
	if err := h.projectSvc.Delete(c.Request().Context(), ws.ID, id); err != nil {
		return projectError(c, err)
	}
	return response.Success(c, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ProjectHandler) ListByTeam(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("teamId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid team ID")
	}
	ws := c.Get("workspace").(*domain.Workspace)
	projects, err := h.projectSvc.ListByTeam(c.Request().Context(), teamID)
	scoped := make([]domain.Project, 0, len(projects))
	for _, p := range projects {
		if p.WorkspaceID == ws.ID {
			scoped = append(scoped, p)
		}
	}
	projects = scoped
	if err != nil {
		return response.InternalError(c)
	}
	statsByID, err := h.projectSvc.GetWorkspaceStats(c.Request().Context(), ws.ID)
	if err != nil {
		return response.InternalError(c)
	}
	resp := make([]dto.ProjectResponse, len(projects))
	for i, p := range projects {
		r := toProjectResponse(p)
		stats := statsByID[p.ID]
		r.Progress = &stats
		resp[i] = r
	}
	return response.Success(c, http.StatusOK, resp)
}

func toProjectResponse(p domain.Project) dto.ProjectResponse {
	resp := dto.ProjectResponse{
		ID:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		SortOrder:   p.SortOrder,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.TeamID != nil {
		s := p.TeamID.String()
		resp.TeamID = &s
	}
	if p.LeadID != nil {
		s := p.LeadID.String()
		resp.LeadID = &s
	}
	resp.StartDate = p.StartDate
	resp.TargetDate = p.TargetDate
	return resp
}

func projectError(c echo.Context, err error) error {
	if errors.Is(err, service.ErrProjectNotFound) {
		return response.NotFound(c, "Project")
	}
	if errors.Is(err, service.ErrInvalidProject) {
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	return response.InternalError(c)
}

func bindProjectRequest(c echo.Context, req any) error {
	decoder := json.NewDecoder(c.Request().Body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 || raw[0] != '{' {
		return errors.New("request must be a JSON object")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request must contain one JSON object")
	}
	fields := json.NewDecoder(bytes.NewReader(raw))
	fields.DisallowUnknownFields()
	return fields.Decode(req)
}
