package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxMessage = 2 << 20
const MaxResponse = 8 << 20

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

func (e *Error) Error() string                    { return e.Code + ": " + e.Message }
func Err(code, message string, status int) *Error { return &Error{code, message, status} }
func ExitCode(err error) int {
	e, ok := err.(*Error)
	if !ok {
		return 1
	}
	switch {
	case e.Status == 401 || e.Status == 403:
		return 3
	case e.Status == 409:
		return 4
	case e.Status == 404:
		return 5
	case e.Status == 429:
		return 6
	case e.Status >= 500 || e.Code == "TRANSPORT_ERROR":
		return 7
	default:
		return 2
	}
}

type Client struct {
	base             *url.URL
	token, workspace string
	http             *http.Client
}

var segment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,199}$`)

func FromEnv() (*Client, error) {
	return New(os.Getenv("SPRINTORIO_URL"), os.Getenv("SPRINTORIO_TOKEN"), os.Getenv("SPRINTORIO_WORKSPACE"))
}
func New(raw, token, workspace string) (*Client, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return nil, Err("CONFIG_ERROR", "SPRINTORIO_URL must be an origin without credentials, query or path", 0)
	}
	ip := net.ParseIP(u.Hostname())
	loopback := u.Hostname() == "localhost" || ip != nil && ip.IsLoopback()
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback) {
		return nil, Err("CONFIG_ERROR", "Use HTTPS or loopback HTTP", 0)
	}
	if !strings.HasPrefix(token, "spr_") || strings.ContainsAny(token, "\r\n\t ") || len(token) > 1024 {
		return nil, Err("CONFIG_ERROR", "SPRINTORIO_TOKEN must contain an agent token", 0)
	}
	if !segment.MatchString(workspace) || workspace == "." || workspace == ".." {
		return nil, Err("CONFIG_ERROR", "SPRINTORIO_WORKSPACE must contain a workspace slug", 0)
	}
	return &Client{base: u, token: token, workspace: workspace, http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func DecodeInput(raw []byte) (map[string]any, error) {
	if len(raw) > MaxMessage {
		return nil, Err("INPUT_TOO_LARGE", "Input exceeds 2 MiB", 0)
	}
	var m map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&m); err != nil || m == nil {
		return nil, Err("INVALID_INPUT", "Expected one JSON object", 0)
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, Err("INVALID_INPUT", "Expected one JSON object", 0)
	}
	return m, nil
}
func (c *Client) Call(ctx context.Context, name string, input map[string]any) (any, error) {
	op, ok := Lookup(name)
	if !ok {
		return nil, Err("UNKNOWN_OPERATION", "Use discover to select an operation", 0)
	}
	schema, _ := Schema(name)
	if err := validate(input, schema.(map[string]any)); err != nil {
		return nil, err
	}
	fields := []string{}
	if f, ok := input["fields"].([]any); ok {
		for _, v := range f {
			fields = append(fields, v.(string))
		}
	}
	for _, f := range fields {
		if !safeFields[f] {
			return nil, Err("INVALID_INPUT", "Unknown or excluded output field", 0)
		}
	}
	path := op.Path
	for _, p := range op.Params {
		v, _ := input[p].(string)
		if !segment.MatchString(v) || v == "." || v == ".." {
			return nil, Err("INVALID_INPUT", "Invalid path identifier: "+p, 0)
		}
		path = strings.ReplaceAll(path, ":"+p, v)
	}
	u := *c.base
	u.Path = "/api/workspaces/" + c.workspace + path
	q := url.Values{}
	if query, ok := input["query"].(map[string]any); ok {
		for k, v := range query {
			q.Set(k, valueString(v))
		}
	}
	if name == "issues.list" || name == "machines.list" || name == "runs.list" || name == "machines.agent_runs" {
		if q.Get("page") == "" {
			q.Set("page", "1")
		}
		if q.Get("per_page") == "" {
			q.Set("per_page", "25")
		}
	}

	if name == "machines.events" || name == "machines.logs" {
		if q.Get("limit") == "" {
			limit := "100"
			if name == "machines.logs" {
				limit = "250"
			}
			q.Set("limit", limit)
		}
	}
	u.RawQuery = q.Encode()
	if op.Kind == "authenticated_download" {
		return map[string]any{"download": map[string]any{"url": u.String(), "authentication": "Bearer agent token", "max_bytes": MaxTransfer}, "hint": "Download with your own authenticated HTTP client; workspace ZIP: sprintorio export --output FILE"}, nil
	}
	if name == "assets.upload" {
		return c.uploadBase64(ctx, input)
	}
	var body []byte
	if op.Body != nil {
		body, _ = json.Marshal(input["body"])
		if len(body) > MaxMessage {
			return nil, Err("INPUT_TOO_LARGE", "Input exceeds 2 MiB", 0)
		}
	}
	req, err := http.NewRequestWithContext(ctx, op.Method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, Err("INVALID_INPUT", "Invalid request", 0)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if op.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, Err("TRANSPORT_ERROR", "API request failed or timed out", 0)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponse+1))
	if err != nil {
		return nil, Err("TRANSPORT_ERROR", "Could not read API response", 0)
	}
	if len(raw) > MaxResponse {
		return nil, Err("RESPONSE_TOO_LARGE", "API response exceeds 8 MiB; narrow the query", 0)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, httpError(resp.StatusCode)
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "application/zip") {
		return nil, Err("UNSUPPORTED_MEDIA_TYPE", "ZIP transfers require the web export flow; JSON tools cannot return archives", 0)
	}
	if len(raw) == 0 {
		return map[string]any{"ok": true}, nil
	}
	var result any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&result) != nil || d.Decode(new(any)) != io.EOF {
		return nil, Err("INVALID_RESPONSE", "API did not return one JSON value", 0)
	}
	if strings.HasPrefix(name, "shared_links.") {
		result = omitURL(result)
	}
	detail, _ := input["detail"].(bool)

	if (name == "delivery.get" || name == "delivery.update") && !detail && len(fields) == 0 {
		if m, ok := result.(map[string]any); ok {
			return map[string]any{"version": m["version"], "updated_at": m["updated_at"], "plan_summary": project(m["plan"], false, nil, c.token), "hint": "Read delivery.get with detail:true before replacing the full plan"}, nil
		}
	}

	projected := project(result, detail, fields, c.token)
	if name == "machines.events" || name == "machines.logs" {
		if rows, ok := result.([]any); ok {
			var cursor any = json.Number(q.Get("after_id"))
			if q.Get("after_id") == "" {
				cursor = json.Number("0")
			}
			if len(rows) > 0 {
				if last, ok := rows[len(rows)-1].(map[string]any); ok {
					cursor = last["id"]
				}
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			return map[string]any{"data": projected, "pagination": map[string]any{"next_after_id": cursor, "limit": limit, "may_have_more": len(rows) == limit}}, nil
		}
	}
	return projected, nil
}
func valueString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return ""
	}
}
func httpError(status int) *Error {
	switch status {
	case 401:
		return Err("UNAUTHORIZED", "Agent token invalid, expired or revoked", status)
	case 403:
		return Err("FORBIDDEN", "Token scope or workspace role denies this operation", status)
	case 404:
		return Err("NOT_FOUND", "Resource not found", status)
	case 409:
		return Err("CONFLICT", "Resource changed; read current state before updating", status)
	case 429:
		return Err("RATE_LIMITED", "Rate limit reached; retry later", status)
	default:
		return Err("API_ERROR", "API rejected the request", status)
	}
}
func validate(v any, s map[string]any) error {
	typ := s["type"]
	if typ == nil {
		return nil
	}
	types := []string{}
	switch x := typ.(type) {
	case string:
		types = []string{x}
	case []string:
		types = x
	}
	matched := ""
	for _, t := range types {
		switch t {
		case "null":
			if v == nil {
				matched = t
			}
		case "object":
			if _, ok := v.(map[string]any); ok {
				matched = t
			}
		case "array":
			if _, ok := v.([]any); ok {
				matched = t
			}
		case "string":
			if _, ok := v.(string); ok {
				matched = t
			}
		case "boolean":
			if _, ok := v.(bool); ok {
				matched = t
			}
		case "number", "integer":
			_, isJSONNumber := v.(json.Number)
			_, isFloat := v.(float64)
			n, err := strconv.ParseFloat(valueString(v), 64)
			if (isJSONNumber || isFloat) && err == nil && (t == "number" || n == float64(int64(n))) {
				matched = t
			}
		}
	}
	if matched == "" {
		return Err("INVALID_INPUT", "Input field has an incorrect type", 0)
	}
	if matched == "object" {
		m := v.(map[string]any)
		p, constrained := s["properties"].(map[string]any)
		if !constrained {
			return nil
		}
		if req, ok := s["required"].([]string); ok {
			for _, k := range req {
				if _, ok := m[k]; !ok {
					return Err("INVALID_INPUT", "Required field missing: "+k, 0)
				}
			}
		}
		for k, x := range m {
			child, ok := p[k]
			if !ok {
				return Err("INVALID_INPUT", "Unknown input field", 0)
			}
			if err := validate(x, child.(map[string]any)); err != nil {
				return err
			}
		}
	}
	if matched == "array" {
		a := v.([]any)
		if min, ok := s["minItems"].(int); ok && len(a) < min {
			return Err("INVALID_INPUT", "Too few array items", 0)
		}
		if max, ok := s["maxItems"].(int); ok && len(a) > max {
			return Err("INVALID_INPUT", "Too many array items", 0)
		}
		for _, x := range a {
			if err := validate(x, s["items"].(map[string]any)); err != nil {
				return err
			}
		}
	}
	if matched == "integer" || matched == "number" {
		n, _ := strconv.ParseFloat(valueString(v), 64)
		if min, ok := s["minimum"].(int); ok && n < float64(min) {
			return Err("INVALID_INPUT", "Number below allowed minimum", 0)
		}
		if max, ok := s["maximum"].(int); ok && n > float64(max) {
			return Err("INVALID_INPUT", "Number above allowed maximum", 0)
		}
	}
	if matched == "string" {
		str := v.(string)
		if max, ok := s["maxLength"].(int); ok && utf8.RuneCountInString(str) > max {
			return Err("INVALID_INPUT", "String exceeds allowed length", 0)
		}
		if min, ok := s["minLength"].(int); ok && utf8.RuneCountInString(str) < min {
			return Err("INVALID_INPUT", "String is too short", 0)
		}
		if enum, ok := s["enum"].([]string); ok {
			found := false
			for _, e := range enum {
				if e == str {
					found = true
				}
			}
			if !found {
				return Err("INVALID_INPUT", "Value outside allowed enum", 0)
			}
		}
	}
	return nil
}

var safeFields = fieldSet("logo_url owner_id owner current_user_role share_link_min_role role email filters is_shared url events scope scope_id include_description expires_at provider base_url model has_api_key description_expand_prompt default_prompt default_issue_copy_prompt machine_id size services repo owner base_branch working_branch keep_running environment_id available policy idle_timeout_minutes max_machines resource_usage cpu memory disk repositories runs trace event level message timestamp state metadata github_repo_id scope_type installation installation_id account_login account_type repos transitions from_status to_status enabled issue_count total_issues distribution insights burnup issue links commit pull_requests checked_out_at checkout_id format exported_at source_schema_version warnings requires_reconfiguration id identifier name title key description status status_id status_info priority team_id project_id cycle_id creator_id assignee_id assignee_ids parent_id due_date sort_order labels creator assignee assignees parent sub_issue_count sub_issue_done relation_counts relation_summary is_subscribed created_at updated_at issue_id user_id field old_value new_value old_display_value new_display_value related blocked_by blocking duplicate category color position slug is_default project_ids start_date target_date lead_id progress total completed cancelled number goals retrospective end_date completed_at recurrence_rule next_run_at is_active created_by workspace_id label_ids body resolved_at user replies related_issue_id type related_issue display_name avatar_url filename content_type asset_id size entity_type entity_id runtime_session_name maximum_sessions triage_enabled parent_auto_close_enabled sub_issue_auto_close_enabled issue_copy_prompt icon data total_count page per_page has_more ok plan version product_name objective success_metric target_release milestones test_cases due_date steps expected_result evidence")
var summaryFields = fieldSet("id identifier name title key status status_id priority team_id project_id cycle_id assignee_id parent_id due_date target_date category position slug is_default project_ids number start_date end_date is_active related_issue_id type resolved_at field created_at updated_at version progress total completed cancelled total_count page per_page has_more ok data plan milestones test_cases product_name target_release")

func fieldSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, v := range strings.Fields(s) {
		m[v] = true
	}
	return m
}
func project(v any, detail bool, fields []string, token string) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, v := range x {
			if !safeFields[k] {
				continue
			}
			include := detail || summaryFields[k]
			if len(fields) > 0 {
				include = k == "data" || k == "plan" || k == "total_count" || k == "page" || k == "per_page" || k == "has_more" || k == "version"
				for _, f := range fields {
					if f == k {
						include = true
					}
				}
			}
			if include {
				nestedFields := []string(nil)
				if k == "data" {
					nestedFields = fields
				}
				if k == "filters" || k == "recurrence_rule" || k == "services_config" || k == "metadata" || k == "payload" {
					out[k] = sanitizeDynamic(v, token)
				} else {
					out[k] = project(v, detail, nestedFields, token)
				}
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = project(v, detail, fields, token)
		}
		return out
	case string:
		return credentialPattern.ReplaceAllString(strings.ReplaceAll(x, token, "[redacted]"), "[redacted]")
	default:
		return v
	}
}
func (c *Client) Context(ctx context.Context) (any, error) {
	out := map[string]any{"workspace": c.workspace}
	for _, r := range []string{"teams", "labels", "projects"} {
		v, err := c.Call(ctx, r+".list", map[string]any{})
		if err != nil {
			return nil, err
		}
		out[r] = v
	}
	return out, nil
}

var credentialPattern = regexp.MustCompile(`spr_[A-Za-z0-9_-]+`)

func sanitizeDynamic(v any, token string) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, v := range x {
			lower := strings.ToLower(k)
			if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || lower == "api_key" {
				continue
			}
			out[k] = sanitizeDynamic(v, token)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = sanitizeDynamic(v, token)
		}
		return out
	case string:
		return credentialPattern.ReplaceAllString(strings.ReplaceAll(x, token, "[redacted]"), "[redacted]")
	default:
		return v
	}
}

func omitURL(v any) any {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "url")
	case []any:
		for _, v := range x {
			omitURL(v)
		}
	}
	return v
}
