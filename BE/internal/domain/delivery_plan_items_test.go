package domain

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeliveryReadinessRequiresRecordedPassingEvidence(t *testing.T) {
	require.False(t, (DeliveryPlan{}).Readiness().Ready)
	plan := DeliveryPlan{ProductName: "Product", Objective: "Ship", SuccessMetric: "Adoption", Milestones: []DeliveryMilestone{{ID: "m", Title: "Ship", Status: "planned"}}, TestCases: []DeliveryTestCase{{ID: "t", Title: "Login", Steps: "Open", ExpectedResult: "Dashboard", Status: "blocked", Evidence: "Offline"}}}
	result := plan.Readiness()
	require.False(t, result.Ready)
	require.Equal(t, "saved_plan_and_test_evidence", result.Basis)
	require.Equal(t, []DeliveryAttention{{Kind: "milestone", ID: "m", Title: "Ship", Status: "planned"}, {Kind: "test_case", ID: "t", Title: "Login", Status: "blocked"}}, result.Attention)
	plan.Milestones[0].Status = "done"
	plan.TestCases[0].Status = "passed"
	plan.TestCases[0].Evidence = " "
	require.False(t, plan.Readiness().Ready)
	plan.TestCases[0].Evidence = "Actual smoke run"
	require.True(t, plan.Readiness().Ready)
}

func TestDeliveryItemChangesDoNotMutateSourceOnFailureAndEnforceFinalLimit(t *testing.T) {
	original := DeliveryPlan{TestCases: make([]DeliveryTestCase, 200)}
	for i := range original.TestCases {
		original.TestCases[i] = DeliveryTestCase{ID: string(rune(1000 + i)), Title: "Case", Status: "not_run"}
	}
	_, err := (DeliveryPlanItems{TestCases: &DeliveryTestCaseChanges{Upsert: []DeliveryTestCase{{ID: "extra", Title: "Extra", Status: "not_run"}}}}).Apply(original)
	require.Error(t, err)
	require.Len(t, original.TestCases, 200)
	updated := original.TestCases[0]
	updated.Title = "Changed"
	_, err = (DeliveryPlanItems{TestCases: &DeliveryTestCaseChanges{Upsert: []DeliveryTestCase{updated, {ID: "invalid", Title: "", Status: "not_run"}}}}).Apply(original)
	require.Error(t, err)
	require.Equal(t, "Case", original.TestCases[0].Title)
}

func TestDeliveryReadinessBoundsAttentionWithoutDroppingCounts(t *testing.T) {
	plan := DeliveryPlan{ProductName: "P", Objective: "O", SuccessMetric: "M"}
	for i := 0; i < 25; i++ {
		plan.TestCases = append(plan.TestCases, DeliveryTestCase{ID: string(rune(1000 + i)), Title: "Test", Status: "not_run"})
	}
	result := plan.Readiness()
	require.Len(t, result.Attention, 20)
	require.Equal(t, 5, result.AttentionOmitted)
	require.Equal(t, 25, result.TestCases["not_run"])
}
