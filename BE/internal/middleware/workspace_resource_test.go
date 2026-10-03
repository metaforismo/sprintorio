package middleware_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/handler"
	mw "github.com/metaforismo/sprintorio/BE/internal/middleware"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	jwtpkg "github.com/metaforismo/sprintorio/BE/pkg/jwt"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceResourceScopePostgres(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not configured")
	}
	db, err := sqlx.Connect("pgx", url)
	require.NoError(t, err)
	defer db.Close()
	user, other, ws, foreign := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, u := range []uuid.UUID{user, other} {
		_, err = db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Scope tester','unusable')`, u, u.String()+"@example.test")
		require.NoError(t, err)
	}
	defer func() {
		db.Exec(`DELETE FROM workspaces WHERE id=$1 OR id=$2`, ws, foreign)
		db.Exec(`DELETE FROM users WHERE id=$1 OR id=$2`, user, other)
	}()
	for _, w := range []uuid.UUID{ws, foreign} {
		_, err = db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Scope workspace',$2,$3)`, w, "scope-"+w.String(), user)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, w, user)
		require.NoError(t, err)
	}
	svc := service.NewAgentTokenService(repository.NewAgentTokenRepository(db))
	full, err := svc.Create(context.Background(), ws, user, service.CreateAgentTokenRequest{Name: "Full scope tester", Scope: "full"})
	require.NoError(t, err)
	jwt, err := jwtpkg.GenerateAccessToken(user, "scope-secret")
	require.NoError(t, err)
	e := echo.New()
	group := e.Group("/api/workspaces/:slug", mw.AuthWithAgentTokens("scope-secret", svc), mw.WorkspaceMembership(repository.NewWorkspaceRepository(db)), mw.AgentTokenWorkspaceAccess(), mw.WorkspaceResourceScope(repository.NewWorkspaceResourceRepository(db)))
	ok := func(c echo.Context) error { return c.NoContent(204) }
	for _, resource := range []string{"issue-templates", "shared-links", "webhooks"} {
		group.GET("/"+resource+"/:id", ok)
		group.PATCH("/"+resource+"/:id", ok)
		group.DELETE("/"+resource+"/:id", ok)
	}
	viewH := handler.NewViewHandler(service.NewViewService(repository.NewViewRepository(db)))
	group.POST("/views", viewH.Create, mw.RequirePermission(domain.PermViewManage))
	group.GET("/views/:id", viewH.Get)
	group.PATCH("/views/:id", viewH.Update, mw.RequirePermission(domain.PermViewManage))
	group.DELETE("/views/:id", viewH.Delete, mw.RequirePermission(domain.PermViewManage))
	base := "/api/workspaces/scope-" + ws.String()
	call := func(method, path, token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, base+path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r)
		return w
	}
	for _, resource := range []string{"issue-templates", "shared-links", "webhooks"} {
		ownID, foreignID := uuid.New(), uuid.New()
		for _, record := range []struct{ id, w uuid.UUID }{{ownID, ws}, {foreignID, foreign}} {
			switch resource {
			case "issue-templates":
				_, err = db.Exec(`INSERT INTO issue_templates(id,workspace_id,title,created_by) VALUES($1,$2,'Scope template',$3)`, record.id, record.w, user)
			case "shared-links":
				_, err = db.Exec(`INSERT INTO shared_links(id,workspace_id,created_by,token,scope) VALUES($1,$2,$3,$4,'workspace')`, record.id, record.w, user, uuid.NewString())
			case "webhooks":
				_, err = db.Exec(`INSERT INTO webhooks(id,workspace_id,url,secret) VALUES($1,$2,'https://example.test','synthetic')`, record.id, record.w)
			}
			require.NoError(t, err)
		}
		for _, token := range []string{jwt, full.Token} {
			for _, method := range []string{"GET", "PATCH", "DELETE"} {
				require.Equal(t, 404, call(method, "/"+resource+"/"+foreignID.String(), token, "{}").Code)
				require.Equal(t, 204, call(method, "/"+resource+"/"+ownID.String(), token, "{}").Code)
			}
		}
	}
	ownView, foreignView, privateView, sharedView := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, v := range []struct {
		id, w, u uuid.UUID
		shared   bool
	}{{ownView, ws, user, false}, {foreignView, foreign, user, true}, {privateView, ws, other, false}, {sharedView, ws, other, true}} {
		_, err = db.Exec(`INSERT INTO views(id,workspace_id,creator_id,name,filters,is_shared) VALUES($1,$2,$3,'Scope view','{}',$4)`, v.id, v.w, v.u, v.shared)
		require.NoError(t, err)
	}

	// Scoped share publication rejects foreign resources and another user's private view.
	shares := service.NewSharedLinkService(repository.NewSharedLinkRepository(db), repository.NewWorkspaceRepository(db), repository.NewTeamRepository(db), repository.NewProjectRepository(db), repository.NewViewRepository(db), nil, nil, nil, "scope-secret")
	for _, id := range []uuid.UUID{foreignView, privateView} {
		raw := id.String()
		_, err = shares.Create(context.Background(), ws, user, dto.CreateSharedLinkRequest{Scope: "view", ScopeID: &raw})
		require.Error(t, err)
	}
	raw := ownView.String()
	_, err = shares.Create(context.Background(), ws, user, dto.CreateSharedLinkRequest{Scope: "view", ScopeID: &raw})
	require.NoError(t, err)
	// Validate template references on both creation and update before persistence.
	ownTeam, foreignTeam, ownLabel, foreignLabel := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, v := range []struct{ id, w uuid.UUID }{{ownTeam, ws}, {foreignTeam, foreign}} {
		_, err = db.Exec(`INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Template team','TPL')`, v.id, v.w)
		require.NoError(t, err)
	}
	for _, v := range []struct{ id, w uuid.UUID }{{ownLabel, ws}, {foreignLabel, foreign}} {
		_, err = db.Exec(`INSERT INTO labels(id,workspace_id,name,color) VALUES($1,$2,'Template label','#abcdef')`, v.id, v.w)
		require.NoError(t, err)
	}
	templateRepo := repository.NewIssueTemplateRepository(db)
	tmpl := &domain.IssueTemplate{ID: uuid.New(), WorkspaceID: ws, CreatedBy: user, Title: "Reference test", TeamID: &foreignTeam, LabelIDs: json.RawMessage(`[]`), IsActive: true}
	require.Error(t, templateRepo.Create(context.Background(), tmpl))
	tmpl.TeamID = &ownTeam
	tmpl.AssigneeID = &other
	require.Error(t, templateRepo.Create(context.Background(), tmpl))
	tmpl.AssigneeID = &user
	tmpl.LabelIDs = json.RawMessage(`["` + foreignLabel.String() + `"]`)
	require.Error(t, templateRepo.Create(context.Background(), tmpl))
	tmpl.LabelIDs = json.RawMessage(`["` + strings.ToUpper(ownLabel.String()) + `"]`)
	require.NoError(t, templateRepo.Create(context.Background(), tmpl))
	tmpl.TeamID = &foreignTeam
	require.Error(t, templateRepo.Update(context.Background(), tmpl))
	stored, err := templateRepo.GetByID(context.Background(), tmpl.ID)
	require.NoError(t, err)
	require.Equal(t, ownTeam, *stored.TeamID)
	for _, token := range []string{jwt, full.Token} {
		require.Equal(t, 404, call("GET", "/views/"+foreignView.String(), token, "").Code)
		require.Equal(t, 404, call("GET", "/views/"+privateView.String(), token, "").Code)
		require.Equal(t, 200, call("GET", "/views/"+sharedView.String(), token, "").Code)
		require.Equal(t, 404, call("PATCH", "/views/"+sharedView.String(), token, `{"name":"Denied"}`).Code)
	}
	// Normal view lifecycle remains usable and all mutations stay inside the workspace.
	created := call("POST", "/views", jwt, `{"name":"Normal view","filters":{},"is_shared":false}`)
	require.Equal(t, 201, created.Code)
	var view struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &view))
	require.Equal(t, 200, call("GET", "/views/"+view.ID, jwt, "").Code)
	require.Equal(t, 200, call("PATCH", "/views/"+view.ID, jwt, `{"name":"Updated view"}`).Code)
	require.Equal(t, 200, call("DELETE", "/views/"+view.ID, jwt, "").Code)
	_, err = db.Exec(`UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, ws, user)
	require.NoError(t, err)
	require.Equal(t, 403, call("POST", "/views", full.Token, `{"name":"Denied","filters":{}}`).Code)
	require.Equal(t, 403, call("POST", "/views", jwt, `{"name":"Denied","filters":{}}`).Code)
}
