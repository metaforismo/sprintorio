package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/handler"
	mw "github.com/metaforismo/sprintorio/BE/internal/middleware"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	jwtpkg "github.com/metaforismo/sprintorio/BE/pkg/jwt"
	"github.com/stretchr/testify/require"
)

func TestAgentTokensPostgresAndMiddleware(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not configured")
	}
	db, err := sqlx.Connect("pgx", url)
	require.NoError(t, err)
	defer db.Close()
	user, otherUser, ws, otherWS := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{user, otherUser} {
		_, err = db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Token tester','unusable')`, id, id.String()+"@example.test")
		require.NoError(t, err)
	}
	defer func() {
		db.Exec(`DELETE FROM workspaces WHERE id=$1 OR id=$2`, ws, otherWS)
		db.Exec(`DELETE FROM users WHERE id=$1 OR id=$2`, user, otherUser)
	}()
	for _, id := range []uuid.UUID{ws, otherWS} {
		_, err = db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Token workspace',$2,$3)`, id, "token-"+id.String(), user)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, id, user)
		require.NoError(t, err)
	}
	repo := repository.NewAgentTokenRepository(db)
	svc := service.NewAgentTokenService(repo)
	ctx := context.Background()
	created, err := svc.Create(ctx, ws, user, service.CreateAgentTokenRequest{Name: "Local agent"})
	require.NoError(t, err)
	require.True(t, len(created.Token) == 47)
	require.Equal(t, "read", created.Record.Scope)
	var stored string
	require.NoError(t, db.Get(&stored, `SELECT token_hash FROM agent_tokens WHERE id=$1`, created.Record.ID))
	require.True(t, stored == service.AgentTokenHash(created.Token))
	require.False(t, stored == created.Token)
	encoded, err := json.Marshal(created.Record)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(encoded), stored))
	require.False(t, strings.Contains(string(encoded), created.Token))
	for _, req := range []service.CreateAgentTokenRequest{{Name: ""}, {Name: "Valid", Scope: "admin"}, {Name: "Valid", ExpiresInDays: new(int)}} {
		_, err := svc.Create(ctx, ws, user, req)
		require.ErrorIs(t, err, service.ErrInvalidAgentToken)
	}
	revoked, err := svc.Revoke(ctx, created.Record.ID, ws, otherUser)
	require.NoError(t, err)
	require.False(t, revoked)
	e := echo.New()
	api := e.Group("/api", mw.AuthWithAgentTokens("test-secret", svc))
	group := api.Group("/workspaces/:slug", mw.WorkspaceMembership(repository.NewWorkspaceRepository(db)), mw.AgentTokenWorkspaceAccess())
	ok := func(c echo.Context) error { return c.NoContent(204) }
	group.GET("/issues", ok)
	group.GET("/projects/:id/delivery-plan/summary", ok)
	group.PATCH("/projects/:id/delivery-plan/items", ok, mw.RequirePermission("project:manage"))
	group.POST("/views", ok, mw.RequirePermission(domain.PermViewManage))
	group.POST("/issues", ok, mw.RequirePermission(domain.PermIssueCreate))
	group.GET("/members", ok)
	group.POST("/invite", ok, mw.RequirePermission(domain.PermMemberInvite))
	group.PATCH("/ai-settings", ok, mw.RequireOwner())
	group.GET("/export", ok)
	group.GET("/dev-machines", ok)
	group.GET("/issues/:identifier/github", ok)
	group.POST("/issues/:identifier/expand-description", ok)
	group.GET("/new-product-route", ok)
	api.GET("/auth/me", ok)
	h := handler.NewAgentTokenHandler(svc)
	group.GET("/agent-tokens", h.List)
	group.POST("/agent-tokens", h.Create)
	group.DELETE("/agent-tokens/:id", h.Revoke)
	request := func(method, path, bearer, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	base := "/api/workspaces/token-" + ws.String()
	require.Equal(t, 204, request("GET", base+"/issues", created.Token, "").Code)
	var usedBefore, usedAfter string
	require.NoError(t, db.Get(&usedBefore, `SELECT last_used_at::text FROM agent_tokens WHERE id=$1`, created.Record.ID))
	require.Equal(t, 204, request("GET", base+"/issues", created.Token, "").Code)
	require.NoError(t, db.Get(&usedAfter, `SELECT last_used_at::text FROM agent_tokens WHERE id=$1`, created.Record.ID))
	require.Equal(t, usedBefore, usedAfter)
	require.Equal(t, 403, request("POST", base+"/issues", created.Token, "").Code)
	require.Equal(t, 204, request("GET", base+"/projects/test/delivery-plan/summary", created.Token, "").Code)
	require.Equal(t, 403, request("PATCH", base+"/projects/test/delivery-plan/items", created.Token, "").Code)
	require.Equal(t, 403, request("GET", "/api/workspaces/token-"+otherWS.String()+"/issues", created.Token, "").Code)
	for _, path := range []string{"/members", "/export", "/dev-machines", "/agent-tokens", "/issues/ENG-1/github", "/new-product-route"} {
		require.Equal(t, 403, request("GET", base+path, created.Token, "").Code, path)
	}
	require.Equal(t, 403, request("GET", "/api/auth/me", created.Token, "").Code)
	require.Equal(t, 403, request("POST", base+"/agent-tokens", created.Token, `{"name":"Denied"}`).Code)
	require.Equal(t, 403, request("POST", base+"/issues/ENG-1/expand-description", created.Token, "").Code)
	full, err := svc.Create(ctx, ws, user, service.CreateAgentTokenRequest{Name: "Full agent", Scope: "full"})
	require.NoError(t, err)
	require.Equal(t, 204, request("GET", base+"/members", full.Token, "").Code)
	require.Equal(t, 204, request("POST", base+"/invite", full.Token, "").Code)
	require.Equal(t, 204, request("PATCH", base+"/ai-settings", full.Token, "").Code)
	require.Equal(t, 403, request("GET", base+"/agent-tokens", full.Token, "").Code)
	require.Equal(t, 403, request("GET", "/api/auth/me", full.Token, "").Code)
	require.Equal(t, 403, request("GET", "/api/workspaces/token-"+otherWS.String()+"/issues", full.Token, "").Code)
	_, err = db.Exec(`UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, ws, user)
	require.NoError(t, err)
	require.Equal(t, 403, request("POST", base+"/invite", full.Token, "").Code)
	_, err = db.Exec(`UPDATE workspace_members SET role='owner' WHERE workspace_id=$1 AND user_id=$2`, ws, user)
	require.NoError(t, err)
	write, err := svc.Create(ctx, ws, user, service.CreateAgentTokenRequest{Name: "Writer", Scope: "write"})
	require.NoError(t, err)
	require.Equal(t, 204, request("POST", base+"/issues", write.Token, "").Code)
	require.Equal(t, 204, request("PATCH", base+"/projects/test/delivery-plan/items", write.Token, "").Code)
	_, err = db.Exec(`UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, ws, user)
	require.NoError(t, err)
	require.Equal(t, 403, request("POST", base+"/issues", write.Token, "").Code)
	require.Equal(t, 403, request("PATCH", base+"/projects/test/delivery-plan/items", write.Token, "").Code)
	require.Equal(t, 403, request("POST", base+"/views", full.Token, "").Code)
	require.Equal(t, 204, request("GET", base+"/issues", write.Token, "").Code)
	_, err = db.Exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, ws, user)
	require.NoError(t, err)
	require.Equal(t, 401, request("GET", base+"/issues", write.Token, "").Code)
	_, err = db.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, ws, user)
	require.NoError(t, err)
	revoked, err = svc.Revoke(ctx, created.Record.ID, ws, user)
	require.NoError(t, err)
	require.True(t, revoked)
	require.Equal(t, 401, request("GET", base+"/issues", created.Token, "").Code)
	_, err = db.Exec(`UPDATE agent_tokens SET expires_at=NOW()-INTERVAL '1 second' WHERE id=$1`, write.Record.ID)
	require.NoError(t, err)
	require.Equal(t, 401, request("GET", base+"/issues", write.Token, "").Code)
	jwt, err := jwtpkg.GenerateAccessToken(user, "test-secret")
	require.NoError(t, err)
	require.Equal(t, 204, request("GET", "/api/auth/me", jwt, "").Code)
	require.Equal(t, 201, request("POST", base+"/agent-tokens", jwt, `{"name":"JWT created"}`).Code)
	require.Equal(t, 400, request("POST", base+"/agent-tokens", jwt, `{"name":"bad","expires_in_days":0}`).Code)
	require.Equal(t, 200, request("GET", base+"/agent-tokens", jwt, "").Code)
	require.Equal(t, 200, request("DELETE", base+"/agent-tokens/"+write.Record.ID.String(), jwt, "").Code)
	req := httptest.NewRequest(http.MethodGet, base+"/agent-tokens", nil)
	req.Header.Set("Authorization", "Bearer "+created.Token)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: jwt})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)
}
