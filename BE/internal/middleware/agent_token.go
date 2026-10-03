package middleware

import (
	"context"
	"crypto/sha256"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/pkg/response"
)

type AgentTokenAuthenticator interface {
	Authenticate(context.Context, string) (*domain.AgentToken, error)
}

// Exact method and router-pattern allowlist prevents new admin routes inheriting access.
var agentTokenRoutes = map[string]bool{
	"GET /api/workspaces/:slug":                                                 true,
	"GET /api/workspaces/:slug/teams":                                           true,
	"POST /api/workspaces/:slug/teams":                                          true,
	"GET /api/workspaces/:slug/teams/:teamId":                                   true,
	"PATCH /api/workspaces/:slug/teams/:teamId":                                 true,
	"DELETE /api/workspaces/:slug/teams/:teamId":                                true,
	"GET /api/workspaces/:slug/teams/:teamId/statuses":                          true,
	"POST /api/workspaces/:slug/teams/:teamId/statuses":                         true,
	"PATCH /api/workspaces/:slug/teams/:teamId/statuses/:statusId":              true,
	"DELETE /api/workspaces/:slug/teams/:teamId/statuses/:statusId":             true,
	"GET /api/workspaces/:slug/teams/:teamId/cycles":                            true,
	"POST /api/workspaces/:slug/teams/:teamId/cycles":                           true,
	"GET /api/workspaces/:slug/teams/:teamId/cycles/velocity":                   true,
	"GET /api/workspaces/:slug/teams/:teamId/cycles/:cycleId":                   true,
	"PATCH /api/workspaces/:slug/teams/:teamId/cycles/:cycleId":                 true,
	"POST /api/workspaces/:slug/teams/:teamId/cycles/:cycleId/complete":         true,
	"GET /api/workspaces/:slug/teams/:teamId/cycles/:cycleId/burndown":          true,
	"DELETE /api/workspaces/:slug/teams/:teamId/cycles/:cycleId":                true,
	"GET /api/workspaces/:slug/issues":                                          true,
	"POST /api/workspaces/:slug/issues":                                         true,
	"PATCH /api/workspaces/:slug/issues/bulk":                                   true,
	"DELETE /api/workspaces/:slug/issues/bulk":                                  true,
	"GET /api/workspaces/:slug/issues/:identifier":                              true,
	"PATCH /api/workspaces/:slug/issues/:identifier":                            true,
	"DELETE /api/workspaces/:slug/issues/:identifier":                           true,
	"POST /api/workspaces/:slug/issues/:identifier/duplicate":                   true,
	"POST /api/workspaces/:slug/issues/:identifier/convert-to-project":          true,
	"GET /api/workspaces/:slug/issues/:identifier/comments":                     true,
	"POST /api/workspaces/:slug/issues/:identifier/comments":                    true,
	"POST /api/workspaces/:slug/issues/:identifier/comments/:commentId/resolve": true,
	"POST /api/workspaces/:slug/issues/:identifier/comments/:commentId/reopen":  true,
	"GET /api/workspaces/:slug/issues/:identifier/sub-issues":                   true,
	"POST /api/workspaces/:slug/issues/:identifier/sub-issues":                  true,
	"POST /api/workspaces/:slug/issues/:identifier/sub-issues/bulk":             true,
	"GET /api/workspaces/:slug/issues/:identifier/history":                      true,
	"POST /api/workspaces/:slug/issues/:identifier/triage/accept":               true,
	"POST /api/workspaces/:slug/issues/:identifier/triage/decline":              true,
	"POST /api/workspaces/:slug/issues/:identifier/relations":                   true,
	"GET /api/workspaces/:slug/issues/:identifier/relations":                    true,
	"DELETE /api/workspaces/:slug/issues/:identifier/relations/:relationId":     true,
	"GET /api/workspaces/:slug/issue-templates":                                 true,
	"POST /api/workspaces/:slug/issue-templates":                                true,
	"GET /api/workspaces/:slug/issue-templates/:id":                             true,
	"PATCH /api/workspaces/:slug/issue-templates/:id":                           true,
	"DELETE /api/workspaces/:slug/issue-templates/:id":                          true,
	"GET /api/workspaces/:slug/labels":                                          true,
	"POST /api/workspaces/:slug/labels":                                         true,
	"PATCH /api/workspaces/:slug/labels/:id":                                    true,
	"DELETE /api/workspaces/:slug/labels/:id":                                   true,
	"GET /api/workspaces/:slug/projects":                                        true,
	"POST /api/workspaces/:slug/projects":                                       true,
	"GET /api/workspaces/:slug/projects/:id":                                    true,
	"GET /api/workspaces/:slug/projects/:id/delivery-plan":                      true,
	"PATCH /api/workspaces/:slug/projects/:id/delivery-plan":                    true,
	"PATCH /api/workspaces/:slug/projects/:id":                                  true,
	"DELETE /api/workspaces/:slug/projects/:id":                                 true,
	"GET /api/workspaces/:slug/teams/:teamId/projects":                          true,
	"GET /api/workspaces/:slug/views":                                           true,
	"POST /api/workspaces/:slug/views":                                          true,
	"GET /api/workspaces/:slug/views/:id":                                       true,
	"PATCH /api/workspaces/:slug/views/:id":                                     true,
	"DELETE /api/workspaces/:slug/views/:id":                                    true,
}

type agentRateBucket struct {
	at     time.Time
	tokens float64
}
type agentRateGate struct {
	mu      sync.Mutex
	buckets map[[32]byte]agentRateBucket
}

func (g *agentRateGate) allow(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	hash := sha256.Sum256([]byte(key))
	bucket, ok := g.buckets[hash]
	if !ok {
		if len(g.buckets) >= 4096 {
			for k, v := range g.buckets {
				if now.Sub(v.at) > time.Minute {
					delete(g.buckets, k)
				}
			}
			if len(g.buckets) >= 4096 {
				return false
			}
		}
		bucket = agentRateBucket{at: now, tokens: 40}
	}
	bucket.tokens = min(40, bucket.tokens+now.Sub(bucket.at).Seconds()*20)
	bucket.at = now
	allowed := bucket.tokens >= 1
	if allowed {
		bucket.tokens--
	}
	g.buckets[hash] = bucket
	return allowed
}

// AuthWithAgentTokens preserves JWT auth and accepts personal tokens only in Bearer headers.
func AuthWithAgentTokens(secret string, auth AgentTokenAuthenticator) echo.MiddlewareFunc {
	gate := &agentRateGate{buckets: make(map[[32]byte]agentRateBucket)}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		jwtHandler := Auth(secret)(next)
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || !strings.HasPrefix(parts[1], "spr_") {
				return jwtHandler(c)
			}
			if len(header) > 128 {
				return response.Unauthorized(c)
			}
			if !gate.allow("ip:"+c.RealIP()) || !gate.allow("token:"+parts[1]) {
				c.Response().Header().Set("Retry-After", "1")
				return response.Error(c, 429, "RATE_LIMITED", "Too many requests")
			}
			if !strings.HasPrefix(c.Path(), "/api/workspaces/:slug") || c.Param("slug") == "" || strings.HasPrefix(c.Path(), "/api/workspaces/:slug/agent-tokens") {
				return response.Forbidden(c)
			}
			token, err := auth.Authenticate(c.Request().Context(), parts[1])
			if err != nil || token == nil {
				return response.Unauthorized(c)
			}
			if token.Scope != "full" && !agentTokenRoutes[c.Request().Method+" "+c.Path()] {
				return response.Forbidden(c)
			}
			if token.Scope == "read" && c.Request().Method != http.MethodGet {
				return response.Forbidden(c)
			}
			c.Set(string(UserIDKey), token.UserID)
			c.Set("agent_token", token)
			return next(c)
		}
	}
}

// Membership and resource middleware still check current user role and resource ownership.
func AgentTokenWorkspaceAccess() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, ok := c.Get("agent_token").(*domain.AgentToken)
			if !ok {
				return next(c)
			}
			ws := GetWorkspace(c)
			role := GetWorkspaceRole(c)
			if ws == nil || token.WorkspaceID != ws.ID || !domain.HasPermission(role, domain.PermIssueRead) {
				return response.Forbidden(c)
			}
			if c.Request().Method != http.MethodGet && strings.HasPrefix(c.Path(), "/api/workspaces/:slug/views") && !domain.HasPermission(role, domain.PermViewManage) {
				return response.Forbidden(c)
			}
			return next(c)
		}
	}
}
