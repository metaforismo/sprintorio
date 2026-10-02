package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type DeliveryPlan struct {
	ProductName   string              `json:"product_name"`
	Objective     string              `json:"objective"`
	SuccessMetric string              `json:"success_metric"`
	TargetRelease string              `json:"target_release"`
	Milestones    []DeliveryMilestone `json:"milestones"`
	TestCases     []DeliveryTestCase  `json:"test_cases"`
}
type DeliveryMilestone struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	DueDate string `json:"due_date"`
	Status  string `json:"status"`
}
type DeliveryTestCase struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Steps          string `json:"steps"`
	ExpectedResult string `json:"expected_result"`
	Status         string `json:"status"`
	Evidence       string `json:"evidence"`
}

func (p *DeliveryPlan) Normalize() {
	if p.Milestones == nil {
		p.Milestones = []DeliveryMilestone{}
	}
	if p.TestCases == nil {
		p.TestCases = []DeliveryTestCase{}
	}
}
func (p DeliveryPlan) Validate() error {
	for _, field := range []struct {
		name, value string
		max         int
	}{
		{"product_name", p.ProductName, 200}, {"objective", p.Objective, 5000}, {"success_metric", p.SuccessMetric, 2000},
	} {
		if utf8.RuneCountInString(field.value) > field.max {
			return fmt.Errorf("%s exceeds %d characters", field.name, field.max)
		}
	}
	if !deliveryDate(p.TargetRelease) {
		return fmt.Errorf("target_release must be YYYY-MM-DD or empty")
	}
	if len(p.Milestones) > 200 || len(p.TestCases) > 200 {
		return fmt.Errorf("milestones and test_cases are limited to 200 entries each")
	}
	ids := map[string]bool{}
	for _, m := range p.Milestones {
		if !deliveryID(m.ID, ids) {
			return fmt.Errorf("milestone IDs must be nonempty, unique and at most 100 characters")
		}
		if strings.TrimSpace(m.Title) == "" || utf8.RuneCountInString(m.Title) > 300 {
			return fmt.Errorf("milestone title must contain 1 to 300 characters")
		}
		if !deliveryDate(m.DueDate) {
			return fmt.Errorf("milestone due_date must be YYYY-MM-DD or empty")
		}
		if m.Status != "planned" && m.Status != "in_progress" && m.Status != "done" {
			return fmt.Errorf("invalid milestone status")
		}
	}
	ids = map[string]bool{}
	for _, tc := range p.TestCases {
		if !deliveryID(tc.ID, ids) {
			return fmt.Errorf("test case IDs must be nonempty, unique and at most 100 characters")
		}
		if strings.TrimSpace(tc.Title) == "" || utf8.RuneCountInString(tc.Title) > 300 {
			return fmt.Errorf("test case title must contain 1 to 300 characters")
		}
		if utf8.RuneCountInString(tc.Steps) > 10000 || utf8.RuneCountInString(tc.ExpectedResult) > 5000 || utf8.RuneCountInString(tc.Evidence) > 10000 {
			return fmt.Errorf("test case steps, expected_result or evidence exceeds its length limit")
		}
		if tc.Status != "not_run" && tc.Status != "passed" && tc.Status != "failed" && tc.Status != "blocked" {
			return fmt.Errorf("invalid test case status")
		}
		if (tc.Status == "passed" || tc.Status == "failed") && strings.TrimSpace(tc.Evidence) == "" {
			return fmt.Errorf("passed and failed tests require evidence")
		}
	}
	return nil
}
func deliveryDate(s string) bool {
	if s == "" {
		return true
	}
	t, err := time.Parse("2006-01-02", s)
	return err == nil && t.Format("2006-01-02") == s
}
func deliveryID(s string, ids map[string]bool) bool {
	if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 100 || ids[s] {
		return false
	}
	ids[s] = true
	return true
}
