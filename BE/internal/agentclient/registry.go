// Package agentclient provides the shared, finite operation surface for CLI and MCP.
package agentclient

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
)

type Operation struct {
	Kind   string   `json:"kind,omitempty"`
	Name   string   `json:"name"`
	Method string   `json:"method"`
	Path   string   `json:"-"`
	Params []string `json:"params,omitempty"`
	Query  []string `json:"query,omitempty"`
	Body   any      `json:"-"`
}
type planUpdate struct {
	Plan    domain.DeliveryPlan `json:"plan" validate:"required"`
	Version int                 `json:"version" validate:"required,min=0"`
}

var operations = buildRegistry()

func buildRegistry() map[string]Operation {
	r := map[string]Operation{}
	add := func(name, method, path string, body any, query ...string) {
		op := Operation{Name: name, Method: method, Path: path, Body: body, Query: query}
		for _, part := range strings.Split(path, "/") {
			if strings.HasPrefix(part, ":") {
				op.Params = append(op.Params, part[1:])
			}
		}
		r[name] = op
	}
	resource := func(name, path string, create, update any, get bool) {
		add(name+".list", "GET", path, nil)
		if get {
			add(name+".get", "GET", path+"/:id", nil)
		}
		if create != nil {
			add(name+".create", "POST", path, create)
		}
		if update != nil {
			add(name+".update", "PATCH", path+"/:id", update)
		}
	}
	resource("issues", "/issues", dto.CreateIssueRequest{}, dto.UpdateIssueRequest{}, true)
	op := r["issues.list"]
	op.Query = []string{"page", "per_page", "status", "status_type", "priority", "assignee", "creator", "team", "project", "cycle", "label", "search", "due_before", "due_after", "triaged", "sub_issues", "parent_id", "sort", "order"}
	r[op.Name] = op
	resource("projects", "/projects", dto.CreateProjectRequest{}, dto.UpdateProjectRequest{}, true)
	resource("teams", "/teams", dto.CreateTeamRequest{}, dto.UpdateTeamRequest{}, true)
	resource("statuses", "/teams/:team_id/statuses", dto.CreateTeamStatusRequest{}, dto.UpdateTeamStatusRequest{}, false)
	resource("cycles", "/teams/:team_id/cycles", dto.CreateCycleRequest{}, dto.UpdateCycleRequest{}, true)
	resource("labels", "/labels", dto.CreateLabelRequest{}, dto.UpdateLabelRequest{}, false)
	resource("templates", "/issue-templates", dto.CreateIssueTemplateRequest{}, dto.UpdateIssueTemplateRequest{}, true)
	resource("comments", "/issues/:issue/comments", dto.CreateCommentRequest{}, nil, false)
	resource("relations", "/issues/:issue/relations", dto.CreateIssueRelationRequest{}, nil, false)
	add("relations.delete", "DELETE", "/issues/:issue/relations/:id", nil)
	add("comments.resolve", "POST", "/issues/:issue/comments/:id/resolve", nil)
	add("comments.reopen", "POST", "/issues/:issue/comments/:id/reopen", nil)
	add("history.list", "GET", "/issues/:issue/history", nil)
	add("issues.subissues", "GET", "/issues/:id/sub-issues", nil)
	add("issues.create_subissue", "POST", "/issues/:id/sub-issues", dto.CreateSubIssueRequest{})
	add("cycles.complete", "POST", "/teams/:team_id/cycles/:id/complete", dto.CompleteCycleRequest{})
	add("delivery.get", "GET", "/projects/:id/delivery-plan", nil)
	add("delivery.update", "PATCH", "/projects/:id/delivery-plan", planUpdate{})
	// Workspace administrative operations retain backend role checks and full scope.
	add("workspace.get", "GET", "", nil)
	add("workspace.update", "PATCH", "", dto.UpdateWorkspaceRequest{})
	add("workspace.delete", "DELETE", "", nil)
	add("workspace.export", "GET", "/export", nil)
	add("members.list", "GET", "/members", nil)
	add("members.invite", "POST", "/invite", dto.InviteMemberRequest{})
	add("members.update", "PATCH", "/members/:id", dto.UpdateMemberRoleRequest{})
	add("members.delete", "DELETE", "/members/:id", nil)
	for _, name := range []string{"issues", "projects", "teams", "labels", "templates", "cycles", "statuses"} {
		op := r[name+".update"]
		add(name+".delete", "DELETE", op.Path, nil)
	}
	add("teams.leave", "POST", "/teams/:id/leave", nil)
	add("issues.bulk_update", "PATCH", "/issues/bulk", dto.BulkUpdateIssueRequest{})
	add("issues.bulk_delete", "DELETE", "/issues/bulk", dto.BulkDeleteIssueRequest{})
	add("issues.bulk_create_subissues", "POST", "/issues/:id/sub-issues/bulk", dto.BulkCreateSubIssueRequest{})
	add("issues.duplicate", "POST", "/issues/:id/duplicate", dto.DuplicateIssueRequest{})
	add("issues.convert_to_project", "POST", "/issues/:id/convert-to-project", nil)
	add("issues.subscribe", "POST", "/issues/:id/subscribe", nil)
	add("issues.unsubscribe", "DELETE", "/issues/:id/subscribe", nil)
	add("issues.triage_accept", "POST", "/issues/:id/triage/accept", nil)
	add("issues.triage_decline", "POST", "/issues/:id/triage/decline", nil)
	add("issues.expand_description", "POST", "/issues/:id/expand-description", dto.ExpandIssueDescriptionRequest{})
	add("cycles.velocity", "GET", "/teams/:team_id/cycles/velocity", nil)
	add("cycles.burndown", "GET", "/teams/:team_id/cycles/:id/burndown", nil)
	add("projects.by_team", "GET", "/teams/:team_id/projects", nil)
	resource("views", "/views", dto.CreateViewRequest{}, dto.UpdateViewRequest{}, true)
	add("views.delete", "DELETE", "/views/:id", nil)
	resource("webhooks", "/webhooks", dto.CreateWebhookRequest{}, dto.UpdateWebhookRequest{}, false)
	add("webhooks.delete", "DELETE", "/webhooks/:id", nil)
	resource("shared_links", "/shared-links", dto.CreateSharedLinkRequest{}, dto.UpdateSharedLinkRequest{}, false)
	add("shared_links.delete", "DELETE", "/shared-links/:id", nil)
	resource("favorites", "/favorites", dto.CreateFavoriteRequest{}, nil, false)
	add("favorites.delete", "DELETE", "/favorites/:id", nil)
	add("ai_settings.get", "GET", "/ai-settings", nil)
	add("ai_settings.update", "PATCH", "/ai-settings", dto.UpdateAISettingsRequest{})
	add("ai_settings.issue_copy_prompt", "GET", "/ai-settings/issue-copy-prompt", nil)
	for _, suffix := range []string{"overview", "distribution", "insights", "burnup"} {
		add("analytics."+suffix, "GET", "/analytics/"+suffix, nil)
	}
	for _, suffix := range []string{"status", "repos", "auto-transitions", "issue-links"} {
		add("github."+strings.ReplaceAll(suffix, "-", "_"), "GET", "/github/"+suffix, nil)
	}
	add("github.link_repos", "POST", "/github/repos", dto.LinkGitHubReposRequest{})
	add("github.unlink_repo", "DELETE", "/github/repos/:id", nil)
	add("github.update_transitions", "PATCH", "/github/auto-transitions", dto.UpdateAutoTransitionsRequest{})
	add("github.disconnect", "DELETE", "/github/disconnect", nil)
	add("github.issue_activity", "GET", "/issues/:id/github", nil)
	resource("machines", "/dev-machines", dto.CreateDevMachineRequest{}, dto.UpdateDevMachineRequest{}, true)
	add("machines.delete", "DELETE", "/dev-machines/:id", nil)
	add("machines.bulk_delete", "DELETE", "/dev-machines/bulk", dto.BulkDeleteDevMachinesRequest{})
	for _, suffix := range []string{"start", "stop", "pause", "teardown", "activity"} {
		add("machines."+suffix, "POST", "/dev-machines/:id/"+suffix, nil)
	}
	for _, suffix := range []string{"checkouts", "events", "logs", "services", "providers", "resource-usage", "agent-runs"} {
		add("machines."+strings.ReplaceAll(suffix, "-", "_"), "GET", "/dev-machines/:id/"+suffix, nil)
	}
	add("machines.checkout", "POST", "/dev-machines/:id/checkouts", dto.CheckoutIssueRequest{})
	add("machine_policy.get", "GET", "/dev-machine-policy", nil)
	add("machine_policy.update", "PATCH", "/dev-machine-policy", dto.DevMachinePolicyRequest{})
	add("machine_providers.list", "GET", "/dev-machine-providers", nil)
	add("machine_scope.list", "GET", "/dev-machine-scope-settings", nil)
	add("machine_scope.get", "GET", "/dev-machine-scope-setting", nil, "scope_type", "scope_id")
	add("machine_scope.update", "PUT", "/dev-machine-scope-setting", dto.DevMachineScopeSettingRequest{})
	add("machine_scope.delete", "DELETE", "/dev-machine-scope-setting", nil, "scope_type", "scope_id")
	add("environments.list", "GET", "/dev-machine-environments", nil)
	add("environments.get", "GET", "/dev-machine-environments/:id", nil)
	add("environments.create", "POST", "/dev-machine-environments", dto.CreateDevMachineEnvironmentRequest{})
	add("environments.delete", "DELETE", "/dev-machine-environments/:id", nil)
	add("runs.list", "GET", "/agent-runs", nil)
	add("runs.get", "GET", "/agent-runs/:id", nil)
	add("runs.cancel", "POST", "/agent-runs/:id/cancel", nil)
	add("runs.trace", "GET", "/agent-runs/:id/trace", nil, "events_after_id", "events_limit", "logs_after_id", "logs_limit")

	add("runs.create", "POST", "/dev-machines/:id/agent-runs", dto.CreateAgentRunRequest{})
	add("services.launch", "POST", "/dev-machines/:id/services/:service/launch", nil)
	add("terminals.list", "GET", "/dev-machines/:id/terminal-sessions", nil)
	add("terminals.create", "POST", "/dev-machines/:id/terminal-sessions", dto.CreateTerminalSessionRequest{})
	add("terminals.close", "POST", "/dev-machines/:id/terminal-sessions/:session_id/close", nil)
	add("assets.get", "GET", "/assets/:id", nil)
	for _, name := range []string{"analytics.overview", "analytics.distribution"} {
		op := r[name]
		op.Query = []string{"team_id"}
		r[name] = op
	}
	for _, name := range []string{"analytics.insights", "analytics.burnup"} {
		op := r[name]
		op.Query = []string{"measure", "slice", "segment", "from", "to", "interval", "team_id", "project_id", "cycle_id", "assignee_id", "creator_id", "status_id", "status_type", "priority", "label_id", "include_sub_issues", "include_triage"}
		r[name] = op
	}
	add("assets.upload", "POST", "/upload", uploadInput{})
	for _, name := range []string{"workspace.export", "assets.get"} {
		op := r[name]
		op.Kind = "authenticated_download"
		r[name] = op
	}
	opUpload := r["assets.upload"]
	opUpload.Kind = "base64_upload_1MiB"
	r[opUpload.Name] = opUpload
	add("machines.permanent_delete", "POST", "/dev-machines/:id/permanent-delete", nil)
	add("machines.bulk_permanent_delete", "POST", "/dev-machines/bulk/permanent-delete", dto.BulkDeleteDevMachinesRequest{})
	add("machine_names.suggestion", "GET", "/dev-machine-names/suggestion", nil)
	add("machine_names.availability", "GET", "/dev-machine-names/availability", nil, "name")
	for _, name := range []string{"machines.list", "runs.list", "machines.agent_runs"} {
		op := r[name]
		op.Query = []string{"page", "per_page"}
		if name == "machines.list" {
			op.Query = append(op.Query, "status", "issue_id")
		}
		r[name] = op
	}
	for _, name := range []string{"machines.events", "machines.logs"} {
		op := r[name]
		op.Query = []string{"after_id", "limit"}
		if name == "machines.logs" {
			op.Query = append(op.Query, "agent_run_id")
		}
		r[name] = op
	}
	return r
}
func Discover(resource string) []Operation {
	out := []Operation{}
	for name, op := range operations {
		if resource == "" || strings.HasPrefix(name, resource+".") {
			out = append(out, op)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func Lookup(name string) (Operation, bool) { op, ok := operations[name]; return op, ok }
func ObjectSchema(properties map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}
func Schema(name string) (any, error) {
	op, ok := Lookup(name)
	if !ok {
		return nil, Err("UNKNOWN_OPERATION", "Use discover to select an operation", 0)
	}
	p := map[string]any{}
	required := []string{}
	for _, k := range op.Params {
		p[k] = map[string]any{"type": "string", "minLength": 1}
		required = append(required, k)
	}
	if len(op.Query) > 0 {
		q := map[string]any{}
		for _, k := range op.Query {
			if k == "page" || k == "per_page" || k == "limit" || k == "after_id" || strings.HasSuffix(k, "_limit") || strings.HasSuffix(k, "_after_id") {
				q[k] = map[string]any{"type": "integer", "minimum": 1}
				if k == "per_page" {
					q[k].(map[string]any)["maximum"] = 100
				}
				if k == "after_id" || strings.HasSuffix(k, "_after_id") {
					q[k].(map[string]any)["minimum"] = 0
				}
				if k == "limit" {
					max := 200
					if name == "machines.logs" {
						max = 1000
					}
					q[k].(map[string]any)["maximum"] = max
				}
				if k == "events_limit" {
					q[k].(map[string]any)["maximum"] = 500
				}
				if k == "logs_limit" {
					q[k].(map[string]any)["maximum"] = 2000
				}
			} else {
				if k == "include_sub_issues" || k == "include_triage" {
					q[k] = map[string]any{"type": "boolean"}
				} else {
					q[k] = map[string]any{"type": "string"}
				}
			}
		}
		p["query"] = ObjectSchema(q)
	}
	if op.Body != nil {
		p["body"] = typeSchema(reflect.TypeOf(op.Body))
		if name == "delivery.update" {
			bodyProperties := p["body"].(map[string]any)["properties"].(map[string]any)
			bodyProperties["version"].(map[string]any)["minimum"] = 0
			planProperties := bodyProperties["plan"].(map[string]any)["properties"].(map[string]any)
			for key, max := range map[string]int{"product_name": 200, "objective": 5000, "success_metric": 2000} {
				planProperties[key].(map[string]any)["maxLength"] = max
			}
			for _, key := range []string{"milestones", "test_cases"} {
				array := planProperties[key].(map[string]any)
				array["maxItems"] = 200
				item := array["items"].(map[string]any)
				item["required"] = []string{"id", "title", "status"}
				props := item["properties"].(map[string]any)
				props["id"].(map[string]any)["minLength"] = 1
				props["id"].(map[string]any)["maxLength"] = 100
				props["title"].(map[string]any)["minLength"] = 1
				props["title"].(map[string]any)["maxLength"] = 300
				if key == "milestones" {
					props["status"].(map[string]any)["enum"] = []string{"planned", "in_progress", "done"}
				} else {
					props["status"].(map[string]any)["enum"] = []string{"not_run", "passed", "failed", "blocked"}
					props["evidence"].(map[string]any)["description"] = "Required and nonempty when passed or failed"
					for field, max := range map[string]int{"steps": 10000, "expected_result": 5000, "evidence": 10000} {
						props[field].(map[string]any)["maxLength"] = max
					}
				}
			}
		}
		required = append(required, "body")
	}
	p["detail"] = map[string]any{"type": "boolean"}
	p["fields"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": 40}
	return ObjectSchema(p, required...), nil
}
func typeSchema(t reflect.Type) map[string]any {
	nullable := t.Kind() == reflect.Pointer
	if nullable {
		t = t.Elem()
	}
	var s map[string]any
	if t == reflect.TypeOf(dto.OptionalString{}) {
		s = map[string]any{"type": []string{"string", "null"}}
		return s
	}
	if t == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{}
	}
	switch t.Kind() {
	case reflect.Interface:
		return map[string]any{}
	case reflect.Map:
		return map[string]any{"type": "object"}
	case reflect.Struct:
		p := map[string]any{}
		req := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			fs := typeSchema(f.Type)
			tag := f.Tag.Get("validate")
			if strings.HasPrefix(tag, "required") {
				req = append(req, name)
			}
			for _, v := range strings.Split(tag, ",") {
				if v == "dive" {
					break
				}
				for _, bound := range []string{"min=", "max="} {
					if strings.HasPrefix(v, bound) {
						n, e := strconv.Atoi(strings.TrimPrefix(v, bound))
						if e == nil {
							kind := f.Type.Kind()
							if kind == reflect.Pointer {
								kind = f.Type.Elem().Kind()
							}
							key := "minimum"
							if bound == "max=" {
								key = "maximum"
							}
							if kind == reflect.String {
								key = "minLength"
								if bound == "max=" {
									key = "maxLength"
								}
							}
							if kind == reflect.Slice {
								key = "minItems"
								if bound == "max=" {
									key = "maxItems"
								}
							}
							fs[key] = n
						}
					}
				}
				if v == "uuid" {
					fs["format"] = "uuid"
				}
				if strings.HasPrefix(v, "oneof=") {
					fs["enum"] = strings.Fields(strings.TrimPrefix(v, "oneof="))
				}
			}
			p[name] = fs
		}
		s = ObjectSchema(p, req...)
	case reflect.Slice:
		s = map[string]any{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Bool:
		s = map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64:
		s = map[string]any{"type": "integer"}
	case reflect.Float64:
		s = map[string]any{"type": "number"}
	default:
		s = map[string]any{"type": "string"}
	}
	if nullable {
		if typ, ok := s["type"].(string); ok {
			s["type"] = []string{typ, "null"}
		}
	}
	return s
}
