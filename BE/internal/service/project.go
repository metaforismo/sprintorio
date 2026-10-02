package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/metaforismo/sprintorio/BE/pkg/sanitize"
	"github.com/metaforismo/sprintorio/BE/pkg/validate"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrInvalidProject = errors.New("invalid project")

func invalidProject(message string) error { return fmt.Errorf("%w: %s", ErrInvalidProject, message) }

func (s *ProjectService) validateReferences(ctx context.Context, p *domain.Project) error {
	if p.TeamID == nil && p.LeadID == nil {
		return nil
	}
	valid, err := s.projectRepo.ValidateReferences(ctx, p.WorkspaceID, p.TeamID, p.LeadID)
	if err != nil {
		return err
	}
	if !valid {
		return invalidProject("team and lead must belong to this workspace")
	}
	return nil
}
func projectDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *value)
	if err != nil || t.Format("2006-01-02") != *value {
		return nil, invalidProject("dates must be YYYY-MM-DD or null")
	}
	return &t, nil
}
func projectDatesValid(p *domain.Project) error {
	if p.StartDate != nil && p.TargetDate != nil && p.TargetDate.Before(*p.StartDate) {
		return invalidProject("target_date must be on or after start_date")
	}
	return nil
}
func projectUUID(value *string) (*uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	id, err := uuid.Parse(*value)
	if err != nil || id == uuid.Nil || validate.V.Var(*value, "uuid") != nil {
		return nil, invalidProject("team_id and lead_id must be valid UUIDs or null")
	}
	return &id, nil
}

type ProjectService struct {
	projectRepo repository.ProjectRepo
}

func NewProjectService(projectRepo repository.ProjectRepo) *ProjectService {
	return &ProjectService{projectRepo: projectRepo}
}

func (s *ProjectService) Create(ctx context.Context, workspaceID uuid.UUID, req dto.CreateProjectRequest) (*domain.Project, error) {
	if err := validate.Struct(&req); err != nil {
		return nil, invalidProject("invalid project fields")
	}
	req.Name = strings.TrimSpace(sanitize.StripHTML(req.Name))
	if req.Name == "" {
		return nil, invalidProject("name must not be blank")
	}
	if req.Description != nil {
		clean := sanitize.SanitizeHTML(*req.Description)
		req.Description = &clean
	}

	project := &domain.Project{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Description: req.Description,
		Status:      domain.ProjectStatusPlanned,
	}
	var err error
	project.TeamID, err = projectUUID(req.TeamID)
	if err != nil {
		return nil, err
	}
	project.LeadID, err = projectUUID(req.LeadID)
	if err != nil {
		return nil, err
	}
	project.StartDate, err = projectDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	project.TargetDate, err = projectDate(req.TargetDate)
	if err != nil {
		return nil, err
	}
	if err := projectDatesValid(project); err != nil {
		return nil, err
	}
	if err := s.validateReferences(ctx, project); err != nil {
		return nil, err
	}
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return s.projectRepo.GetByID(ctx, id)
}

func (s *ProjectService) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Project, error) {
	return s.projectRepo.ListByWorkspace(ctx, workspaceID)
}

func (s *ProjectService) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]domain.Project, error) {
	return s.projectRepo.ListByTeam(ctx, teamID)
}

func (s *ProjectService) Update(ctx context.Context, workspaceID, id uuid.UUID, req dto.UpdateProjectRequest) (*domain.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	if project.WorkspaceID != workspaceID {
		return nil, ErrProjectNotFound
	}

	if err := validate.Struct(&req); err != nil {
		return nil, invalidProject("invalid project fields")
	}
	copy := *project
	project = &copy
	if req.Name != nil {
		clean := strings.TrimSpace(sanitize.StripHTML(*req.Name))
		if clean == "" {
			return nil, invalidProject("name must not be blank")
		}
		req.Name = &clean
		project.Name = clean
	}
	if req.Description != nil {
		clean := sanitize.SanitizeHTML(*req.Description)
		req.Description = &clean
		project.Description = req.Description
	}
	if req.Status != nil {
		project.Status = domain.ProjectStatus(*req.Status)
	}
	if req.TeamID.Set {
		project.TeamID, err = projectUUID(req.TeamID.Value)
		if err != nil {
			return nil, err
		}
	}
	if req.LeadID.Set {
		project.LeadID, err = projectUUID(req.LeadID.Value)
		if err != nil {
			return nil, err
		}
	}
	if req.StartDate.Set {
		project.StartDate, err = projectDate(req.StartDate.Value)
		if err != nil {
			return nil, err
		}
	}
	if req.TargetDate.Set {
		project.TargetDate, err = projectDate(req.TargetDate.Value)
		if err != nil {
			return nil, err
		}
	}
	if req.SortOrder != nil {
		if math.IsNaN(*req.SortOrder) || math.IsInf(*req.SortOrder, 0) {
			return nil, invalidProject("invalid sort_order")
		}
		project.SortOrder = *req.SortOrder
	}
	if err := projectDatesValid(project); err != nil {
		return nil, err
	}
	// Validate associations supplied by this PATCH. A formerly valid lead may
	// have left the workspace; that must not block unrelated metadata edits.
	references := &domain.Project{WorkspaceID: workspaceID}
	if req.TeamID.Set {
		references.TeamID = project.TeamID
	}
	if req.LeadID.Set {
		references.LeadID = project.LeadID
	}
	if err := s.validateReferences(ctx, references); err != nil {
		return nil, err
	}

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if project == nil {
		return ErrProjectNotFound
	}
	if project.WorkspaceID != workspaceID {
		return ErrProjectNotFound
	}
	return s.projectRepo.Delete(ctx, id)
}

func (s *ProjectService) GetStats(ctx context.Context, projectID uuid.UUID) (*dto.ProjectProgressResponse, error) {
	total, completed, cancelled, err := s.projectRepo.IssueStats(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &dto.ProjectProgressResponse{
		Total:     total,
		Completed: completed,
		Cancelled: cancelled,
	}, nil
}

func (s *ProjectService) GetWorkspaceStats(ctx context.Context, workspaceID uuid.UUID) (map[uuid.UUID]dto.ProjectProgressResponse, error) {
	return s.projectRepo.IssueStatsByWorkspace(ctx, workspaceID)
}
