package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestDeliveryPlanItemChangesAreAtomicAndPreserveUnrelatedRecords(t *testing.T) {
	ws, id := uuid.New(), uuid.New()
	original := domain.DeliveryPlan{ProductName: "Product", Objective: "Ship", SuccessMetric: "Adoption", Milestones: []domain.DeliveryMilestone{{ID: "launch", Title: "Launch", Status: "planned"}}, TestCases: []domain.DeliveryTestCase{{ID: "login", Title: "Login", Steps: "Open", Status: "passed", Evidence: "Run 1"}, {ID: "keep", Title: "Keep", Status: "not_run"}}}
	raw, err := json.Marshal(original)
	require.NoError(t, err)
	repo := &deliveryPlanFake{project: &domain.Project{ID: id, WorkspaceID: ws, DeliveryPlan: raw}}
	svc := NewProjectService(repo)
	ctx := context.Background()
	edited := original.TestCases[0]
	edited.Steps = "Open again"
	saved, err := svc.UpdateDeliveryPlanItems(ctx, ws, id, domain.DeliveryPlanItems{TestCases: &domain.DeliveryTestCaseChanges{Upsert: []domain.DeliveryTestCase{edited, {ID: "new", Title: "New", Status: "blocked", Evidence: "Service unavailable"}}}}, 0)
	require.NoError(t, err)
	require.Equal(t, 1, saved.Version)
	require.Equal(t, 1, repo.readCount, "compact update must use one scoped read")
	require.Equal(t, "Product", saved.Plan.ProductName)
	require.Equal(t, original.Milestones, saved.Plan.Milestones)
	require.Equal(t, original.TestCases[1], saved.Plan.TestCases[1])
	require.Equal(t, "not_run", saved.Plan.TestCases[0].Status)
	require.Empty(t, saved.Plan.TestCases[0].Evidence)
	_, err = svc.UpdateDeliveryPlanItems(ctx, ws, id, domain.DeliveryPlanItems{TestCases: &domain.DeliveryTestCaseChanges{Remove: []string{"keep"}}}, 0)
	require.ErrorIs(t, err, ErrDeliveryPlanConflict)
	for _, changes := range []domain.DeliveryPlanItems{
		{},
		{TestCases: &domain.DeliveryTestCaseChanges{Remove: []string{"missing"}}},
		{TestCases: &domain.DeliveryTestCaseChanges{Remove: []string{"keep", "keep"}}},
		{TestCases: &domain.DeliveryTestCaseChanges{Upsert: []domain.DeliveryTestCase{{ID: "keep", Title: "Keep", Status: "not_run"}}, Remove: []string{"keep"}}},
		{Milestones: &domain.DeliveryMilestoneChanges{Upsert: []domain.DeliveryMilestone{{ID: "launch", Title: "Done", Status: "done"}}}, TestCases: &domain.DeliveryTestCaseChanges{Upsert: []domain.DeliveryTestCase{{ID: "bad", Title: "Bad", Status: "passed"}}}},
	} {
		_, err = svc.UpdateDeliveryPlanItems(ctx, ws, id, changes, 1)
		require.ErrorIs(t, err, ErrDeliveryPlanValidation)
		require.Equal(t, 1, repo.writeCount)
	}
	persisted, err := svc.GetDeliveryPlan(ctx, ws, id)
	require.NoError(t, err)
	require.Equal(t, saved, persisted)
	saved, err = svc.UpdateDeliveryPlanItems(ctx, ws, id, domain.DeliveryPlanItems{Milestones: &domain.DeliveryMilestoneChanges{Upsert: []domain.DeliveryMilestone{{ID: "launch", Title: "Launch", Status: "done"}}}, TestCases: &domain.DeliveryTestCaseChanges{Remove: []string{"keep", "new"}, Upsert: []domain.DeliveryTestCase{{ID: "login", Title: "Login", Steps: "Open again", ExpectedResult: "Dashboard", Status: "passed", Evidence: "Run 2"}}}}, 1)
	require.NoError(t, err)
	require.Len(t, saved.Plan.TestCases, 1)
	require.Equal(t, "Run 2", saved.Plan.TestCases[0].Evidence)
	summary, err := svc.GetDeliveryPlanSummary(ctx, ws, id)
	require.NoError(t, err)
	require.Equal(t, 2, summary.Version)
	require.True(t, summary.Readiness.Ready)
	require.Empty(t, summary.Readiness.Attention)
	repo.loseRace = true
	_, err = svc.UpdateDeliveryPlanItems(ctx, ws, id, domain.DeliveryPlanItems{TestCases: &domain.DeliveryTestCaseChanges{Remove: []string{"login"}}}, 2)
	require.ErrorIs(t, err, ErrDeliveryPlanConflict)
	_, err = svc.UpdateDeliveryPlanItems(ctx, uuid.New(), id, domain.DeliveryPlanItems{}, 2)
	require.ErrorIs(t, err, ErrDeliveryPlanNotFound)
}
