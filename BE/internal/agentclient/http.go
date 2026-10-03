package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPHandler provides stateless Streamable HTTP with JSON responses and no SSE.
// Every request forwards the caller's agent token, never a shared server credential.
func HTTPHandler(apiURL, workspace string, origins []string) (http.Handler, error) {
	if _, err := New(apiURL, "spr_validation_only", workspace); err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, o := range origins {
		u, e := url.Parse(o)
		if e != nil || strings.Contains(o, "*") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
			return nil, Err("CONFIG_ERROR", "Origins must be exact HTTP(S) origins", 0)
		}
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !allowed[origin] {
			http.Error(w, "Origin denied", http.StatusForbidden)
			return
		}
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.Contains(strings.TrimPrefix(auth, "Bearer "), " ") {
			w.Header().Set("WWW-Authenticate", `Bearer realm="sprintorio"`)
			http.Error(w, "Agent token required", http.StatusUnauthorized)
			return
		}
		client, err := New(apiURL, strings.TrimPrefix(auth, "Bearer "), workspace)
		if err != nil {
			http.Error(w, "Invalid agent token", http.StatusUnauthorized)
			return
		}
		if _, err := client.Call(r.Context(), "workspace.get", map[string]any{}); err != nil {
			status := http.StatusBadGateway
			if e, ok := err.(*Error); ok && (e.Status == 401 || e.Status == 403 || e.Status == 404) {
				status = e.Status
			}
			http.Error(w, "Workspace authorization failed", status)
			return
		}
		contentType, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if mediaErr != nil || contentType != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		if version := r.Header.Get("MCP-Protocol-Version"); version != "" && version != ProtocolVersion && version != "2025-06-18" {
			http.Error(w, "Unsupported MCP protocol version", http.StatusBadRequest)
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxMessage-1))
		if err != nil {
			http.Error(w, "Request exceeds limit or cannot be read", http.StatusRequestEntityTooLarge)
			return
		}
		var req rpcRequest
		if json.Unmarshal(raw, &req) != nil || req.JSONRPC != "2.0" || req.Method == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": nil, "error": map[string]any{"code": -32600, "message": "Invalid Request"}})
			return
		}
		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		// Stateless HTTP clients need no session ID or retained initialized state.
		var compact bytes.Buffer
		if json.Compact(&compact, raw) != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}
		raw = compact.Bytes()
		input := raw
		if req.Method != "initialize" {
			prefix := `{"jsonrpc":"2.0","id":"bootstrap","method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"http","version":"1"}}}` + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
			input = append([]byte(prefix), raw...)
		}
		var result bytes.Buffer
		if err := ServeMCP(r.Context(), bytes.NewReader(append(input, '\n')), &result, client); err != nil {
			http.Error(w, "MCP request failed", http.StatusInternalServerError)
			return
		}
		lines := bytes.Split(bytes.TrimSpace(result.Bytes()), []byte{'\n'})
		if len(lines) == 0 {
			http.Error(w, "MCP response missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(append(lines[len(lines)-1], '\n'))
	}), nil
}
func ServeHTTP(ctx context.Context, listen, apiURL, workspace string, origins []string) error {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return Err("CONFIG_ERROR", "Listen address must be host:port", 0)
	}
	if host == "" {
		return Err("CONFIG_ERROR", "Explicit listen host required; default is loopback", 0)
	}
	handler, err := HTTPHandler(apiURL, workspace, origins)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 50 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return Err("TRANSPORT_ERROR", "HTTP listener failed", 0)
	}
	return nil
}
