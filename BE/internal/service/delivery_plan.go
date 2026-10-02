package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
)

var ErrDeliveryPlanNotFound = errors.New("project not found")
var ErrDeliveryPlanConflict = errors.New("delivery plan has changed; reload before saving")

type DeliveryPlanResponse struct {
	Plan    domain.DeliveryPlan `json:"plan"`
	Version int                 `json:"version"`
}
type deliveryPlanWriter interface {
	UpdateDeliveryPlan(context.Context, uuid.UUID, uuid.UUID, json.RawMessage, int) (bool, error)
}

func (s *ProjectService) GetDeliveryPlan(ctx context.Context, workspaceID, projectID uuid.UUID) (*DeliveryPlanResponse, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil || project.WorkspaceID != workspaceID {
		return nil, ErrDeliveryPlanNotFound
	}
	result := &DeliveryPlanResponse{Version: project.DeliveryPlanVersion}
	if len(project.DeliveryPlan) > 0 {
		if err := json.Unmarshal(project.DeliveryPlan, &result.Plan); err != nil {
			return nil, err
		}
	}
	result.Plan.Normalize()
	return result, nil
}
func (s *ProjectService) UpdateDeliveryPlan(ctx context.Context, workspaceID, projectID uuid.UUID, plan domain.DeliveryPlan, version int) (*DeliveryPlanResponse, error) {
	if version < 0 {
		return nil, errors.New("version must be nonnegative")
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	current, err := s.GetDeliveryPlan(ctx, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if current.Version != version {
		return nil, ErrDeliveryPlanConflict
	}
	plan.Normalize()
	invalidateChangedTestOutcomes(&plan, current.Plan)
	raw, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	writer, ok := s.projectRepo.(deliveryPlanWriter)
	if !ok {
		return nil, errors.New("delivery plan persistence unavailable")
	}
	updated, err := writer.UpdateDeliveryPlan(ctx, workspaceID, projectID, raw, version)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrDeliveryPlanConflict
	}
	return &DeliveryPlanResponse{Plan: plan, Version: version + 1}, nil
}

// An unchanged result describes the old test definition and cannot establish
// that the edited test passed. An explicit new outcome/evidence can be saved.
func invalidateChangedTestOutcomes(plan *domain.DeliveryPlan, previous domain.DeliveryPlan) {
	oldByID := make(map[string]domain.DeliveryTestCase, len(previous.TestCases))
	for _, test := range previous.TestCases {
		oldByID[test.ID] = test
	}
	for i := range plan.TestCases {
		test := &plan.TestCases[i]
		old, ok := oldByID[test.ID]
		if !ok || (test.Status != "passed" && test.Status != "failed" && test.Status != "blocked") {
			continue
		}
		definitionChanged := test.Title != old.Title || test.Steps != old.Steps || test.ExpectedResult != old.ExpectedResult
		if definitionChanged && test.Status == old.Status && test.Evidence == old.Evidence {
			test.Status = "not_run"
			test.Evidence = ""
		}
	}
}
