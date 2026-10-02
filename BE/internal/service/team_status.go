package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/metaforismo/sprintorio/BE/pkg/validate"
	"strings"

	"github.com/google/uuid"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
)

var ErrInvalidTeamStatus = errors.New("invalid team status")
var ErrTeamStatusNotFound = errors.New("status not found")

func invalidTeamStatus(err error) error {
	return fmt.Errorf("%w: %s", ErrInvalidTeamStatus, err.Error())
}

type TeamStatusService struct {
	statusRepo     repository.TeamStatusRepo
	visibilityRepo repository.ProjectStatusVisibilityRepo
}

func NewTeamStatusService(statusRepo repository.TeamStatusRepo, visibilityRepo repository.ProjectStatusVisibilityRepo) *TeamStatusService {
	return &TeamStatusService{statusRepo: statusRepo, visibilityRepo: visibilityRepo}
}

func (s *TeamStatusService) List(ctx context.Context, teamID uuid.UUID) ([]domain.TeamStatus, error) {
	return s.statusRepo.ListByTeam(ctx, teamID)
}

func (s *TeamStatusService) Create(ctx context.Context, teamID uuid.UUID, req dto.CreateTeamStatusRequest) (*domain.TeamStatus, error) {
	if err := validate.Struct(&req); err != nil {
		return nil, invalidTeamStatus(err)
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, invalidTeamStatus(fmt.Errorf("status name must not be blank"))
	}
	projectIDs, err := parseStatusProjectIDs(req.ProjectIDs)
	if err != nil {
		return nil, invalidTeamStatus(err)
	}
	pos, err := s.statusRepo.NextPosition(ctx, teamID)
	if err != nil {
		return nil, err
	}

	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Name), " ", "_"))

	status := &domain.TeamStatus{
		ID:        uuid.New(),
		TeamID:    teamID,
		Name:      req.Name,
		Slug:      slug,
		Category:  domain.StatusCategory(req.Category),
		Color:     req.Color,
		Position:  pos,
		IsDefault: false,
	}

	if err := s.statusRepo.CreateWithProjectVisibility(ctx, status, projectIDs); err != nil {
		if errors.Is(err, repository.ErrStatusProjectScope) {
			return nil, invalidTeamStatus(err)
		}
		return nil, err
	}

	return status, nil
}

func (s *TeamStatusService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateTeamStatusRequest) (*domain.TeamStatus, error) {
	status, err := s.statusRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, ErrTeamStatusNotFound
	}
	copy := *status
	status = &copy

	if err := validate.Struct(&req); err != nil {
		return nil, invalidTeamStatus(err)
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, invalidTeamStatus(fmt.Errorf("status name must not be blank"))
	}
	var projectIDs *[]uuid.UUID
	if req.ProjectIDs != nil {
		ids, err := parseStatusProjectIDs(*req.ProjectIDs)
		if err != nil {
			return nil, invalidTeamStatus(err)
		}
		projectIDs = &ids
	}

	if req.Name != nil {
		status.Name = *req.Name
	}
	if req.Color != nil {
		status.Color = req.Color
	}
	if req.Position != nil {
		status.Position = *req.Position
	}

	if err := s.statusRepo.UpdateWithProjectVisibility(ctx, status, projectIDs); err != nil {
		if errors.Is(err, repository.ErrStatusProjectScope) {
			return nil, invalidTeamStatus(err)
		}
		return nil, err
	}

	return status, nil
}

func (s *TeamStatusService) ListProjectIDsForStatuses(ctx context.Context, statusIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	return s.visibilityRepo.ListProjectIDsByStatuses(ctx, statusIDs)
}

func (s *TeamStatusService) ListProjectsForStatus(ctx context.Context, statusID uuid.UUID) ([]uuid.UUID, error) {
	return s.visibilityRepo.ListProjectsForStatus(ctx, statusID)
}

func (s *TeamStatusService) Delete(ctx context.Context, id uuid.UUID) error {
	status, err := s.statusRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if status == nil {
		return ErrTeamStatusNotFound
	}
	if status.IsDefault {
		return invalidTeamStatus(fmt.Errorf("cannot delete the default status"))
	}
	return s.statusRepo.Delete(ctx, id)
}

func parseStatusProjectIDs(rawIDs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(rawIDs))
	seen := make(map[uuid.UUID]bool)
	for _, raw := range rawIDs {
		id, err := uuid.Parse(raw)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("invalid project_id")
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate project_id")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}
