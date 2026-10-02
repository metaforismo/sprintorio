package service

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/stretchr/testify/require"
	"testing"
)

type deliveryPlanFake struct {
	repository.ProjectRepo
	project    *domain.Project
	writeCount int
	loseRace   bool
}

func (f *deliveryPlanFake) GetByID(context.Context, uuid.UUID) (*domain.Project, error) {
	return f.project, nil
}
func (f *deliveryPlanFake) UpdateDeliveryPlan(_ context.Context, ws, id uuid.UUID, raw json.RawMessage, version int) (bool, error) {
	f.writeCount++
	if f.loseRace || f.project == nil || f.project.WorkspaceID != ws || f.project.ID != id || f.project.DeliveryPlanVersion != version {
		return false, nil
	}
	f.project.DeliveryPlan = raw
	f.project.DeliveryPlanVersion++
	return true, nil
}
func TestDeliveryPlanScopesAndOptimisticConcurrency(t *testing.T) {
	ctx := context.Background()
	ws := uuid.New()
	id := uuid.New()
	repo := &deliveryPlanFake{project: &domain.Project{ID: id, WorkspaceID: ws}}
	svc := NewProjectService(repo)
	initial, err := svc.GetDeliveryPlan(ctx, ws, id)
	require.NoError(t, err)
	require.Equal(t, 0, initial.Version)
	require.NotNil(t, initial.Plan.Milestones)
	_, err = svc.GetDeliveryPlan(ctx, uuid.New(), id)
	require.ErrorIs(t, err, ErrDeliveryPlanNotFound)
	_, err = svc.UpdateDeliveryPlan(ctx, uuid.New(), id, domain.DeliveryPlan{}, 0)
	require.ErrorIs(t, err, ErrDeliveryPlanNotFound)
	require.Zero(t, repo.writeCount)
	saved, err := svc.UpdateDeliveryPlan(ctx, ws, id, domain.DeliveryPlan{Objective: "Release on time"}, 0)
	require.NoError(t, err)
	require.Equal(t, 1, saved.Version)
	_, err = svc.UpdateDeliveryPlan(ctx, ws, id, domain.DeliveryPlan{Objective: "Stale edit"}, 0)
	require.ErrorIs(t, err, ErrDeliveryPlanConflict)
	require.Equal(t, 1, repo.writeCount)
	persisted, err := svc.GetDeliveryPlan(ctx, ws, id)
	require.NoError(t, err)
	require.Equal(t, "Release on time", persisted.Plan.Objective)
	repo.loseRace = true
	_, err = svc.UpdateDeliveryPlan(ctx, ws, id, domain.DeliveryPlan{}, 1)
	require.ErrorIs(t, err, ErrDeliveryPlanConflict)
}

func TestDeliveryPlanInvalidatesStaleTestOutcomes(t *testing.T) {
	old := domain.DeliveryTestCase{ID: "smoke", Title: "Login", Steps: "Open app", ExpectedResult: "Dashboard", Status: "passed", Evidence: "Run 4"}
	for _, field := range []string{"title", "steps", "expected_result"} {
		t.Run(field, func(t *testing.T) {
			test := old
			switch field {
			case "title":
				test.Title = "Changed"
			case "steps":
				test.Steps = "Changed"
			case "expected_result":
				test.ExpectedResult = "Changed"
			}
			plan := domain.DeliveryPlan{TestCases: []domain.DeliveryTestCase{test}}
			invalidateChangedTestOutcomes(&plan, domain.DeliveryPlan{TestCases: []domain.DeliveryTestCase{old}})
			require.Equal(t, "not_run", plan.TestCases[0].Status)
			require.Empty(t, plan.TestCases[0].Evidence)
		})
	}
	fresh := old
	fresh.Steps = "Changed"
	fresh.Evidence = "Run 5"
	plan := domain.DeliveryPlan{TestCases: []domain.DeliveryTestCase{fresh}}
	invalidateChangedTestOutcomes(&plan, domain.DeliveryPlan{TestCases: []domain.DeliveryTestCase{old}})
	require.Equal(t, "passed", plan.TestCases[0].Status)
	require.Equal(t, "Run 5", plan.TestCases[0].Evidence)
}

func TestDeliveryPlanInvalidatesFailedAndBlockedOutcomes(t *testing.T) {
	for _, status := range []string{"failed", "blocked"} {
		previous := domain.DeliveryTestCase{ID: "case", Title: "Old", Status: status, Evidence: "Run 1"}
		changed := previous
		changed.Title = "New"
		plan := domain.DeliveryPlan{TestCases: []domain.DeliveryTestCase{changed}}
		invalidateChangedTestOutcomes(&plan, domain.DeliveryPlan{TestCases: []domain.DeliveryTestCase{previous}})
		require.Equal(t, "not_run", plan.TestCases[0].Status)
		require.Empty(t, plan.TestCases[0].Evidence)
	}
}

func TestDeliveryPlanSaveDoesNotCarryForwardOldPassingEvidence(t *testing.T) {
	ws, id := uuid.New(), uuid.New()
	original := domain.DeliveryPlan{TestCases: []domain.DeliveryTestCase{{ID: "case", Title: "Login", Steps: "Enter password", ExpectedResult: "Dashboard", Status: "passed", Evidence: "Run 1"}}}
	raw, err := json.Marshal(original)
	require.NoError(t, err)
	repo := &deliveryPlanFake{project: &domain.Project{ID: id, WorkspaceID: ws, DeliveryPlan: raw}}
	edited := original
	edited.TestCases = append([]domain.DeliveryTestCase{}, original.TestCases...)
	edited.TestCases[0].Steps = "Enter a different password"
	saved, err := NewProjectService(repo).UpdateDeliveryPlan(context.Background(), ws, id, edited, 0)
	require.NoError(t, err)
	require.Equal(t, "not_run", saved.Plan.TestCases[0].Status)
	require.Empty(t, saved.Plan.TestCases[0].Evidence)
	persisted, err := NewProjectService(repo).GetDeliveryPlan(context.Background(), ws, id)
	require.NoError(t, err)
	require.Equal(t, "not_run", persisted.Plan.TestCases[0].Status)
	require.Empty(t, persisted.Plan.TestCases[0].Evidence)
}
