package repository_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func platformDatabase(t *testing.T) (*sqlx.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	db, err := sqlx.Connect("pgx", url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	user, ws := uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Platform tester','unusable')`, user, user.String()+"@example.test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM workspaces WHERE id=$1`, ws)
		_, _ = db.Exec(`DELETE FROM users WHERE id=$1`, user)
	})
	_, err = db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Platform test',$2,$3)`, ws, "platform-"+ws.String(), user)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, ws, user)
	require.NoError(t, err)
	return db, ws, user
}
func TestAtomicTeamProvisioningAndRollback(t *testing.T) {
	db, ws, user := platformDatabase(t)
	ctx := context.Background()
	repo := repository.NewTeamRepository(db)
	svc := service.NewTeamService(repo, repository.NewTeamStatusRepository(db))
	team, err := svc.Create(ctx, ws, user, dto.CreateTeamRequest{Name: "Engineering", Key: "ENG"})
	require.NoError(t, err)
	var members, statuses int
	require.NoError(t, db.Get(&members, `SELECT COUNT(*) FROM team_members WHERE team_id=$1`, team.ID))
	require.Equal(t, 1, members)
	require.NoError(t, db.Get(&statuses, `SELECT COUNT(*) FROM team_statuses WHERE team_id=$1`, team.ID))
	require.Equal(t, 6, statuses)
	_, err = svc.Create(ctx, ws, user, dto.CreateTeamRequest{Name: "Duplicate", Key: "ENG"})
	require.ErrorIs(t, err, repository.ErrTeamKeyTaken)
	// Failure at the member insert and failure after one status insert must both
	// roll back the team and every already-created child row.
	for _, kind := range []string{"member", "status"} {
		t.Run(kind, func(t *testing.T) {
			id := uuid.New()
			newTeam := &domain.Team{ID: id, WorkspaceID: ws, Name: "Rollback", Key: "ROLL"}
			member := &domain.TeamMember{TeamID: id, UserID: user}
			if kind == "member" {
				member.UserID = uuid.New()
			}
			list := []domain.TeamStatus{{ID: uuid.New(), TeamID: id, Name: "First", Slug: "same", Category: domain.StatusCategoryBacklog}, {ID: uuid.New(), TeamID: id, Name: "Second", Slug: "same", Category: domain.StatusCategoryBacklog}}
			require.Error(t, repo.CreateWithMemberAndStatuses(ctx, newTeam, member, list))
			var count int
			require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM teams WHERE id=$1`, id))
			require.Zero(t, count)
			require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM team_members WHERE team_id=$1`, id))
			require.Zero(t, count)
			require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM team_statuses WHERE team_id=$1`, id))
			require.Zero(t, count)
		})
	}
}
func TestProjectMetadataPersistenceReferencesAndCategoryProgress(t *testing.T) {
	db, ws, user := platformDatabase(t)
	ctx := context.Background()
	teamSvc := service.NewTeamService(repository.NewTeamRepository(db), repository.NewTeamStatusRepository(db))
	team, err := teamSvc.Create(ctx, ws, user, dto.CreateTeamRequest{Name: "Product", Key: "PROD"})
	require.NoError(t, err)
	repo := repository.NewProjectRepository(db)
	svc := service.NewProjectService(repo)
	teamRaw := team.ID.String()
	leadRaw := user.String()
	start, target := "2026-10-02", "2026-10-20"
	p, err := svc.Create(ctx, ws, dto.CreateProjectRequest{Name: "Launch", TeamID: &teamRaw, LeadID: &leadRaw, StartDate: &start, TargetDate: &target})
	require.NoError(t, err)
	loaded, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, start, loaded.StartDate.Format("2006-01-02"))
	require.Equal(t, target, loaded.TargetDate.Format("2006-01-02"))
	_, err = svc.Update(ctx, ws, p.ID, dto.UpdateProjectRequest{StartDate: dto.OptionalString{Set: true}, TeamID: dto.OptionalString{Set: true}, LeadID: dto.OptionalString{Set: true}})
	require.NoError(t, err)
	loaded, err = repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Nil(t, loaded.StartDate)
	require.Nil(t, loaded.TeamID)
	require.Nil(t, loaded.LeadID)
	require.NotNil(t, loaded.TargetDate)
	otherWS := uuid.New()
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM workspaces WHERE id=$1`, otherWS) })
	_, err = db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Other',$2,$3)`, otherWS, otherWS.String(), user)
	require.NoError(t, err)
	otherTeam, err := teamSvc.Create(ctx, otherWS, user, dto.CreateTeamRequest{Name: "Other team", Key: "OTHER"})
	require.NoError(t, err)
	otherRaw := otherTeam.ID.String()
	_, err = svc.Update(ctx, ws, p.ID, dto.UpdateProjectRequest{TeamID: dto.OptionalString{Set: true, Value: &otherRaw}})
	require.ErrorIs(t, err, service.ErrInvalidProject)
	stranger := uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Stranger','unusable')`, stranger, stranger.String()+"@example.test")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id=$1`, stranger) })
	strangerRaw := stranger.String()
	_, err = svc.Update(ctx, ws, p.ID, dto.UpdateProjectRequest{LeadID: dto.OptionalString{Set: true, Value: &strangerRaw}})
	require.ErrorIs(t, err, service.ErrInvalidProject)
	statusID := uuid.New()
	_, err = db.Exec(`INSERT INTO team_statuses(id,team_id,name,slug,category,position) VALUES($1,$2,'Released','released','completed',6)`, statusID, team.ID)
	require.NoError(t, err)
	var cancelledID uuid.UUID
	require.NoError(t, db.Get(&cancelledID, `SELECT id FROM team_statuses WHERE team_id=$1 AND category='cancelled'`, team.ID))
	_, err = db.Exec(`INSERT INTO issues(id,workspace_id,team_id,project_id,number,identifier_text,title,status,status_id,creator_id) VALUES($1,$2,$3,$4,1,'PROD-1','Custom complete','in_progress',$5,$6),($7,$2,$3,$4,2,'PROD-2','Cancelled','cancelled',$8,$6)`, uuid.New(), ws, team.ID, p.ID, statusID, user, uuid.New(), cancelledID)
	require.NoError(t, err)
	stats, err := repo.IssueStatsByWorkspace(ctx, ws)
	require.NoError(t, err)
	require.Equal(t, dto.ProjectProgressResponse{Total: 2, Completed: 1, Cancelled: 1}, stats[p.ID])
	total, completed, cancelled, err := repo.IssueStats(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Equal(t, 1, completed)
	require.Equal(t, 1, cancelled)
	otherStats, err := repo.IssueStatsByWorkspace(ctx, otherWS)
	require.NoError(t, err)
	require.NotContains(t, otherStats, p.ID)
	statusSvc := service.NewTeamStatusService(repository.NewTeamStatusRepository(db), repository.NewProjectStatusVisibilityRepository(db))
	_, err = statusSvc.Create(ctx, otherTeam.ID, dto.CreateTeamStatusRequest{Name: "Foreign scope", Category: "started", ProjectIDs: []string{p.ID.String()}})
	require.ErrorIs(t, err, service.ErrInvalidTeamStatus)
}

func TestStatusVisibilityUpdateRollsBackMetadataAndLinks(t *testing.T) {
	db, ws, user := platformDatabase(t)
	ctx := context.Background()
	team, err := service.NewTeamService(repository.NewTeamRepository(db), repository.NewTeamStatusRepository(db)).Create(ctx, ws, user, dto.CreateTeamRequest{Name: "Team", Key: "QA"})
	require.NoError(t, err)
	projects := repository.NewProjectRepository(db)
	first := &domain.Project{ID: uuid.New(), WorkspaceID: ws, Name: "First", Status: domain.ProjectStatusPlanned}
	second := &domain.Project{ID: uuid.New(), WorkspaceID: ws, Name: "Second", Status: domain.ProjectStatusPlanned}
	require.NoError(t, projects.Create(ctx, first))
	require.NoError(t, projects.Create(ctx, second))
	repo := repository.NewTeamStatusRepository(db)
	svc := service.NewTeamStatusService(repo, repository.NewProjectStatusVisibilityRepository(db))
	status, err := svc.Create(ctx, team.ID, dto.CreateTeamStatusRequest{Name: "Acceptance", Category: "started", ProjectIDs: []string{first.ID.String()}})
	require.NoError(t, err)
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")
	fn := "visibility_fail_" + suffix
	trigger := "visibility_trigger_" + suffix
	_, err = db.Exec(fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture visibility write failure'; END $$`, fn))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON project_status_visibility`, trigger))
		_, _ = db.Exec(fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, fn))
	})
	_, err = db.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON project_status_visibility FOR EACH ROW WHEN (NEW.status_id='%s'::uuid AND NEW.project_id='%s'::uuid) EXECUTE FUNCTION %s()`, trigger, status.ID, second.ID, fn))
	require.NoError(t, err)
	name := "Changed"
	ids := []string{second.ID.String()}
	_, err = svc.Update(ctx, status.ID, dto.UpdateTeamStatusRequest{Name: &name, ProjectIDs: &ids})
	require.Error(t, err)
	loaded, err := repo.GetByID(ctx, status.ID)
	require.NoError(t, err)
	require.Equal(t, "Acceptance", loaded.Name)
	links, err := repository.NewProjectStatusVisibilityRepository(db).ListProjectsForStatus(ctx, status.ID)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{first.ID}, links)
	// Omitting project_ids preserves the existing links; explicit [] clears them.
	_, err = svc.Update(ctx, status.ID, dto.UpdateTeamStatusRequest{Name: &name})
	require.NoError(t, err)
	links, err = repository.NewProjectStatusVisibilityRepository(db).ListProjectsForStatus(ctx, status.ID)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{first.ID}, links)
	empty := []string{}
	_, err = svc.Update(ctx, status.ID, dto.UpdateTeamStatusRequest{ProjectIDs: &empty})
	require.NoError(t, err)
	links, err = repository.NewProjectStatusVisibilityRepository(db).ListProjectsForStatus(ctx, status.ID)
	require.NoError(t, err)
	require.Empty(t, links)
}
