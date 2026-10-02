package domain

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestDeliveryPlanValidation(t *testing.T) {
	valid := DeliveryPlan{ProductName: "Product", TargetRelease: "2026-10-02", Milestones: []DeliveryMilestone{{ID: "m1", Title: "Ship", Status: "planned"}}, TestCases: []DeliveryTestCase{{ID: "t1", Title: "Login", Status: "passed", Evidence: "Run 14: authenticated"}}}
	require.NoError(t, valid.Validate())
	tests := map[string]func(*DeliveryPlan){
		"invalid release":          func(p *DeliveryPlan) { p.TargetRelease = "2026-02-30" },
		"invalid milestone date":   func(p *DeliveryPlan) { p.Milestones[0].DueDate = "2026-1-2" },
		"invalid milestone status": func(p *DeliveryPlan) { p.Milestones[0].Status = "completed" },
		"invalid test status":      func(p *DeliveryPlan) { p.TestCases[0].Status = "done" },
		"empty id":                 func(p *DeliveryPlan) { p.TestCases[0].ID = " " },
		"duplicate ids":            func(p *DeliveryPlan) { p.Milestones = append(p.Milestones, p.Milestones[0]) },
		"empty title":              func(p *DeliveryPlan) { p.TestCases[0].Title = " " },
		"missing evidence":         func(p *DeliveryPlan) { p.TestCases[0].Evidence = " " },
		"failed missing evidence":  func(p *DeliveryPlan) { p.TestCases[0].Status = "failed"; p.TestCases[0].Evidence = "" },
		"long text":                func(p *DeliveryPlan) { p.SuccessMetric = strings.Repeat("x", 2001) },
		"too many tests":           func(p *DeliveryPlan) { p.TestCases = make([]DeliveryTestCase, 201) },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			p := valid
			p.Milestones = append([]DeliveryMilestone{}, valid.Milestones...)
			p.TestCases = append([]DeliveryTestCase{}, valid.TestCases...)
			change(&p)
			require.Error(t, p.Validate())
		})
	}
	empty := DeliveryPlan{}
	empty.Normalize()
	require.NotNil(t, empty.Milestones)
	require.NotNil(t, empty.TestCases)
	require.NoError(t, empty.Validate())
}
