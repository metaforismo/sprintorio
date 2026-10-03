package service

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
)

func TestDeliveryPlanItemsPostgresPreservationAndCAS(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not configured")
	}
	db, err := sqlx.Connect("pgx", url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ws, user, id := uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Delivery items tester','unusable')`, user, user.String()+"@example.test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM workspaces WHERE id=$1`, ws)
		_, _ = db.Exec(`DELETE FROM users WHERE id=$1`, user)
	})
	_, err = db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Items Test',$2,$3)`, ws, "items-"+ws.String(), user)
	require.NoError(t, err)
	repo := repository.NewProjectRepository(db)
	plan := domain.DeliveryPlan{ProductName: "Keep", Milestones: []domain.DeliveryMilestone{{ID: "keep", Title: "Keep", Status: "planned"}}, TestCases: []domain.DeliveryTestCase{{ID: "existing", Title: "Existing", Status: "passed", Evidence: "Run 1"}}}
	raw, err := json.Marshal(plan)
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), &domain.Project{ID: id, WorkspaceID: ws, Name: "Items", Status: domain.ProjectStatusPlanned, DeliveryPlan: raw}))
	svc := NewProjectService(repo)
	changes := domain.DeliveryPlanItems{TestCases: &domain.DeliveryTestCaseChanges{Upsert: []domain.DeliveryTestCase{{ID: "existing", Title: "Edited", Status: "passed", Evidence: "Run 1"}, {ID: "new", Title: "New", Status: "not_run"}}}}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.UpdateDeliveryPlanItems(context.Background(), ws, id, changes, 0)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	successes, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, ErrDeliveryPlanConflict)
			conflicts++
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
	persisted, err := svc.GetDeliveryPlan(context.Background(), ws, id)
	require.NoError(t, err)
	require.Equal(t, 1, persisted.Version)
	require.Equal(t, "Keep", persisted.Plan.ProductName)
	require.Equal(t, plan.Milestones, persisted.Plan.Milestones)
	require.Len(t, persisted.Plan.TestCases, 2)
	require.Equal(t, "not_run", persisted.Plan.TestCases[0].Status)
	require.Empty(t, persisted.Plan.TestCases[0].Evidence)
}
