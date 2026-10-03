package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testToken = "spr_test_credential_do_not_echo"

func fixture(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	c, e := New(s.URL, testToken, "demo")
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestHTTPBoundariesAndProjection(t *testing.T) {
	calls := 0
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer "+testToken || r.URL.Path != "/api/workspaces/demo/issues" || r.URL.Query().Get("per_page") != "25" {
			t.Errorf("unexpected request")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":[{"id":"one","title":"safe","description":"detail %s","password":"never","access_token":"never","status":"todo"}],"page":1,"per_page":25,"total_count":26,"has_more":true}`, testToken)
	})
	v, e := c.Call(context.Background(), "issues.list", map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(v)
	if strings.Contains(string(raw), "description") || strings.Contains(string(raw), "password") || !strings.Contains(string(raw), `"has_more":true`) {
		t.Fatal(string(raw))
	}
	v, e = c.Call(context.Background(), "issues.list", map[string]any{"detail": true, "fields": []any{"id", "description"}})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ = json.Marshal(v)
	if strings.Contains(string(raw), testToken) || strings.Contains(string(raw), "password") || strings.Contains(string(raw), "title") {
		t.Fatal(string(raw))
	}
	for _, input := range []map[string]any{{"query": map[string]any{"per_page": json.Number("101")}}, {"query": map[string]any{"page": "1"}}, {"unknown": true}} {
		if _, e := c.Call(context.Background(), "issues.list", input); e == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if _, e := c.Call(context.Background(), "issues.create", map[string]any{"fields": []any{"password"}, "body": map[string]any{"title": "Secret", "team_id": "team"}}); e == nil {
		t.Fatal("invalid output fields accepted")
	}
	if calls != 2 {
		t.Fatal("invalid input reached API", calls)
	}
}
func TestWritesAndConflict(t *testing.T) {
	calls := 0
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PATCH" {
			t.Error(r.Method)
		}
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		if m["version"] != float64(3) {
			t.Error(m)
		}
		w.WriteHeader(409)
		fmt.Fprintf(w, `{"error":{"message":"%s"}}`, testToken)
	})
	if _, e := c.Call(context.Background(), "delivery.update", map[string]any{"id": "project", "body": map[string]any{"plan": map[string]any{}}}); e == nil {
		t.Fatal("missing version accepted")
	}
	_, e := c.Call(context.Background(), "delivery.update", map[string]any{"id": "project", "body": map[string]any{"version": json.Number("3"), "plan": map[string]any{}}})
	if ExitCode(e) != 4 || strings.Contains(e.Error(), testToken) || calls != 1 {
		t.Fatal(e, calls)
	}
}
func TestURLRedirectAndLimits(t *testing.T) {
	for _, raw := range []string{"http://example.com", "https://u:p@example.com", "https://example.com/api", "https://example.com?token=x"} {
		if _, e := New(raw, testToken, "demo"); e == nil {
			t.Fatal(raw)
		}
	}
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalls++ }))
	defer target.Close()
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) })
	if _, e := c.Call(context.Background(), "teams.list", map[string]any{}); e == nil || targetCalls != 0 {
		t.Fatal("redirect followed")
	}
	c = fixture(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", MaxResponse+1)))
	})
	if _, e := c.Call(context.Background(), "teams.list", map[string]any{}); e == nil || e.(*Error).Code != "RESPONSE_TOO_LARGE" {
		t.Fatal(e)
	}
	if _, e := DecodeInput([]byte(`{} {}`)); e == nil {
		t.Fatal("trailing JSON accepted")
	}
}
func TestMCPStdinLifecycleAndActions(t *testing.T) {
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"id":"p","name":"Project"}`)) })
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":0,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"action","arguments":{"operation":"projects.get","input":{"id":"p"}}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"schema","arguments":{"operation":"delivery.update"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"oops","arguments":{}}}`,
		`not json`}, "\n") + "\n"
	var output bytes.Buffer
	if e := ServeMCP(context.Background(), strings.NewReader(input), &output, c); e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 7 {
		t.Fatal(output.String())
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatal(line)
		}
	}
	if !strings.Contains(lines[0], "-32002") || !strings.Contains(lines[1], ProtocolVersion) || !strings.Contains(lines[3], `"structuredContent"`) || !strings.Contains(lines[5], "-32602") || !strings.Contains(lines[6], "-32700") {
		t.Fatal(output.String())
	}
	if strings.Contains(output.String(), testToken) {
		t.Fatal("credential leak")
	}
	tools, _ := json.Marshal(Tools())
	t.Logf("MCP tool catalogue: %d bytes; %d tools; operations: %d", len(tools), len(Tools()), len(Discover("")))
	if len(tools) > 1600 {
		t.Fatal("catalogue grew")
	}
}
func TestCLIStdinJSONLAndErrorExit(t *testing.T) {
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"a","title":"A"}],"has_more":true,"page":1,"per_page":25,"total_count":30}`))
	})
	t.Setenv("SPRINTORIO_URL", c.base.String())
	t.Setenv("SPRINTORIO_TOKEN", testToken)
	t.Setenv("SPRINTORIO_WORKSPACE", "demo")
	var out, errOut bytes.Buffer
	exit := RunCLI(context.Background(), []string{"call", "issues.list", "--input", "-", "--jsonl"}, strings.NewReader(`{"fields":["id"]}`), &out, &errOut)
	if exit != 0 || strings.Contains(out.String(), "title") || !strings.Contains(out.String(), `"pagination"`) {
		t.Fatal(exit, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	exit = RunCLI(context.Background(), []string{"call", "issues.get", "--input", "-"}, strings.NewReader(`{"id":"../../auth"}`), &out, &errOut)
	if exit != 2 || strings.Contains(errOut.String(), testToken) || out.Len() != 0 {
		t.Fatal(exit, out.String(), errOut.String())
	}
}
func TestHTTPMCPPerCallerAuthenticationAndOrigin(t *testing.T) {
	calls := []string{}
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`{"id":"workspace","name":"Demo"}`))
	})
	if _, e := HTTPHandler(c.base.String(), "demo", []string{"https://*"}); e == nil {
		t.Fatal("wildcard Origin accepted")
	}
	handler, e := HTTPHandler(c.base.String(), "demo", []string{"https://trusted.example"})
	if e != nil {
		t.Fatal(e)
	}
	request := func(token, origin, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://localhost/mcp", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	if w := request("", "", body); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := request(testToken, "https://evil.example", body); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := request("spr_invalid", "", body); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := request(testToken, "https://trusted.example", "{\n  \"jsonrpc\":\"2.0\",\n \"id\":1,\n \"method\":\"tools/list\"\n}"); w.Code != 200 || !strings.Contains(w.Body.String(), `"tools"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(testToken, "", `{"jsonrpc":"2.0","method":"notifications/initialized"}`); w.Code != 202 {
		t.Fatal(w.Code)
	}
	for _, token := range calls {
		if token != "Bearer "+testToken && token != "Bearer spr_invalid" {
			t.Fatal("shared credential used")
		}
	}
}

func TestDynamicFiltersAndTransfer(t *testing.T) {
	c := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/workspaces/demo/views":
			_, _ = w.Write([]byte(`[{"id":"v","filters":{"status_type":["started"],"custom_filter":{"setting":true},"api_key":"do not disclose"}}]`))
		case "/api/workspaces/demo/upload":
			if e := r.ParseMultipartForm(1 << 20); e != nil {
				t.Error(e)
			}
			file, h, e := r.FormFile("file")
			if e != nil {
				t.Error(e)
				return
			}
			defer file.Close()
			if h.Filename != "proof.txt" {
				t.Error(h.Filename)
			}
			_, _ = w.Write([]byte(`{"asset_id":"asset","filename":"proof.txt","content_type":"text/plain","url":"/api/workspaces/demo/assets/asset"}`))
		case "/api/workspaces/demo/export":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write([]byte("PK\x03\x04fixture"))
		}
	})
	v, e := c.Call(context.Background(), "views.list", map[string]any{"detail": true})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(v)
	if !strings.Contains(string(raw), "custom_filter") || strings.Contains(string(raw), "api_key") {
		t.Fatal(string(raw))
	}
	v, e = c.Call(context.Background(), "assets.upload", map[string]any{"body": map[string]any{"filename": "proof.txt", "base64": "cHJvb2Y="}})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ = json.Marshal(v)
	if !strings.Contains(string(raw), "asset_id") {
		t.Fatal(string(raw))
	}
	path := t.TempDir() + "/workspace.zip"
	if _, e = c.Export(context.Background(), path); e != nil {
		t.Fatal(e)
	}
	if _, e = c.Export(context.Background(), path); e == nil {
		t.Fatal("overwrote archive")
	}
	v, e = c.Call(context.Background(), "workspace.export", map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ = json.Marshal(v)
	if strings.Contains(string(raw), testToken) || !strings.Contains(string(raw), "download") {
		t.Fatal(string(raw))
	}
}
