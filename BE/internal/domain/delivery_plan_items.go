package domain

import (
	"fmt"
	"strings"
)

// Item changes replace only the specified IDs. Omitted collections are preserved.
type DeliveryMilestoneChanges struct {
	Upsert []DeliveryMilestone `json:"upsert,omitempty"`
	Remove []string            `json:"remove,omitempty"`
}
type DeliveryTestCaseChanges struct {
	Upsert []DeliveryTestCase `json:"upsert,omitempty"`
	Remove []string           `json:"remove,omitempty"`
}
type DeliveryPlanItems struct {
	Milestones *DeliveryMilestoneChanges `json:"milestones,omitempty"`
	TestCases  *DeliveryTestCaseChanges  `json:"test_cases,omitempty"`
}

func (changes DeliveryPlanItems) Apply(plan DeliveryPlan) (DeliveryPlan, error) {
	count := 0
	if changes.Milestones != nil {
		count += len(changes.Milestones.Upsert) + len(changes.Milestones.Remove)
		items, err := applyDeliveryItems(plan.Milestones, changes.Milestones.Upsert, changes.Milestones.Remove, func(m DeliveryMilestone) string { return m.ID })
		if err != nil {
			return plan, fmt.Errorf("milestones: %w", err)
		}
		plan.Milestones = items
	}
	if changes.TestCases != nil {
		count += len(changes.TestCases.Upsert) + len(changes.TestCases.Remove)
		items, err := applyDeliveryItems(plan.TestCases, changes.TestCases.Upsert, changes.TestCases.Remove, func(t DeliveryTestCase) string { return t.ID })
		if err != nil {
			return plan, fmt.Errorf("test_cases: %w", err)
		}
		plan.TestCases = items
	}
	if count == 0 {
		return plan, fmt.Errorf("at least one item change is required")
	}
	plan.Normalize()
	if err := plan.Validate(); err != nil {
		return plan, err
	}
	return plan, nil
}

func applyDeliveryItems[T any](current, upsert []T, remove []string, id func(T) string) ([]T, error) {
	if len(upsert) > 200 || len(remove) > 200 {
		return nil, fmt.Errorf("upsert and remove are limited to 200 entries each")
	}
	existing := make(map[string]bool, len(current))
	for _, item := range current {
		existing[id(item)] = true
	}
	changed := map[string]bool{}
	replacement := make(map[string]T, len(upsert))
	for _, item := range upsert {
		key := id(item)
		if !deliveryID(key, changed) {
			return nil, fmt.Errorf("change IDs must be nonempty, unique and at most 100 characters")
		}
		replacement[key] = item
	}
	removed := make(map[string]bool, len(remove))
	for _, key := range remove {
		if !deliveryID(key, changed) {
			return nil, fmt.Errorf("change IDs must be nonempty, unique and at most 100 characters")
		}
		if !existing[key] {
			return nil, fmt.Errorf("cannot remove unknown ID %q", key)
		}
		removed[key] = true
	}
	// Copy to keep validation failures from mutating the caller's plan.
	result := make([]T, 0, len(current)+len(upsert))
	for _, item := range current {
		key := id(item)
		if removed[key] {
			continue
		}
		if updated, ok := replacement[key]; ok {
			item = updated
		}
		result = append(result, item)
	}
	for _, item := range upsert {
		if !existing[id(item)] {
			result = append(result, item)
		}
	}
	return result, nil
}

// DeliveryReadiness describes saved records, not execution or release approval.
type DeliveryReadiness struct {
	Basis            string              `json:"basis"`
	Ready            bool                `json:"ready"`
	Milestones       map[string]int      `json:"milestones"`
	TestCases        map[string]int      `json:"test_cases"`
	Attention        []DeliveryAttention `json:"attention"`
	AttentionOmitted int                 `json:"attention_omitted"`
}
type DeliveryAttention struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (p DeliveryPlan) Readiness() DeliveryReadiness {
	result := DeliveryReadiness{Basis: "saved_plan_and_test_evidence", Milestones: map[string]int{"total": len(p.Milestones), "planned": 0, "in_progress": 0, "done": 0}, TestCases: map[string]int{"total": len(p.TestCases), "not_run": 0, "passed": 0, "failed": 0, "blocked": 0}, Attention: []DeliveryAttention{}}

	result.Ready = len(p.TestCases) > 0
	addAttention := func(item DeliveryAttention) {
		result.Ready = false
		if len(result.Attention) < 20 {
			result.Attention = append(result.Attention, item)
		} else {
			result.AttentionOmitted++
		}
	}
	for _, field := range []struct{ name, value string }{{"product_name", p.ProductName}, {"objective", p.Objective}, {"success_metric", p.SuccessMetric}} {
		if strings.TrimSpace(field.value) == "" {
			addAttention(DeliveryAttention{Kind: "brief", ID: field.name, Title: field.name, Status: "missing"})
		}
	}
	for _, m := range p.Milestones {
		result.Milestones[m.Status]++
		if m.Status != "done" {
			addAttention(DeliveryAttention{Kind: "milestone", ID: m.ID, Title: m.Title, Status: m.Status})
		}
	}
	for _, test := range p.TestCases {
		result.TestCases[test.Status]++
		reason := ""
		for _, field := range []string{test.Title, test.Steps, test.ExpectedResult, test.Evidence} {
			if strings.TrimSpace(field) == "" {
				reason = "definition_or_evidence_missing"
				break
			}
		}
		if test.Status != "passed" || reason != "" {
			addAttention(DeliveryAttention{Kind: "test_case", ID: test.ID, Title: test.Title, Status: test.Status, Reason: reason})
		}
	}
	return result
}
