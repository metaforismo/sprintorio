package agentclient

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDeliveryItemsSchemaAndBoundedSummary(t *testing.T) {
	calls := 0
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/api/workspaces/demo/projects/p/delivery-plan/summary":
			if r.Method != http.MethodGet {
				t.Error(r.Method)
			}
			_, _ = w.Write([]byte(`{"version":7,"product_name":"Product","target_release":"","readiness":{"basis":"saved_plan_and_test_evidence","ready":false,"milestones":{"total":1,"planned":1,"in_progress":0,"done":0},"test_cases":{"total":2,"not_run":1,"passed":1,"failed":0,"blocked":0},"attention":[{"kind":"test_case","id":"t","title":"Check","status":"not_run","reason":"definition_or_evidence_missing"}],"attention_omitted":21}}`))
		case "/api/workspaces/demo/projects/p/delivery-plan/items":
			if r.Method != http.MethodPatch {
				t.Error(r.Method)
			}
			var body map[string]any
			if json.NewDecoder(r.Body).Decode(&body) != nil || body["version"] != float64(7) {
				t.Error(body)
			}
			w.WriteHeader(http.StatusConflict)
		default:
			t.Error(r.URL.Path)
		}
	})
	result, err := c.Call(context.Background(), "delivery.summary", map[string]any{"id": "p"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	for _, expected := range []string{`"version":7`, `"basis":"saved_plan_and_test_evidence"`, `"attention_omitted":21`, `"id":"t"`, `"reason":"definition_or_evidence_missing"`, `"passed":1`} {
		if !strings.Contains(string(raw), expected) {
			t.Fatalf("missing %s in %s", expected, raw)
		}
	}
	invalid := []map[string]any{
		{"test_cases": map[string]any{"remove": []any{"t"}}},
		{"version": json.Number("-1"), "test_cases": map[string]any{"remove": []any{"t"}}},
		{"version": json.Number("7"), "test_cases": map[string]any{"upsert": []any{map[string]any{"id": "t", "title": "Test", "status": "invented"}}}},
		{"version": json.Number("7"), "test_cases": map[string]any{"upsert": []any{map[string]any{"id": "t", "status": "not_run"}}}},
	}
	for _, body := range invalid {
		if _, err = c.Call(context.Background(), "delivery.items.update", map[string]any{"id": "p", "body": body}); err == nil {
			t.Fatal("invalid mutation reached server", body)
		}
	}
	_, err = c.Call(context.Background(), "delivery.items.update", map[string]any{"id": "p", "body": map[string]any{"version": json.Number("7"), "test_cases": map[string]any{"remove": []any{"t"}}}})
	if ExitCode(err) != 4 || calls != 2 {
		t.Fatal(err, calls)
	}
}
