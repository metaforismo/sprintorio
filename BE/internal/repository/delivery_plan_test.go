package repository

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
)

func TestDeliveryPlanPersistenceAndConcurrentSave(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	db, err := sqlx.Connect("pgx", url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	ws := uuid.New()
	user := uuid.New()
	id := uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Delivery Tester','unusable')`, user, user.String()+"@example.test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM workspaces WHERE id=$1`, ws)
		_, _ = db.Exec(`DELETE FROM users WHERE id=$1`, user)
	})
	_, err = db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Delivery Test',$2,$3)`, ws, "delivery-"+ws.String(), user)
	require.NoError(t, err)
	repo := NewProjectRepository(db)
	p := &domain.Project{ID: id, WorkspaceID: ws, Name: "Delivery", Status: domain.ProjectStatusPlanned}
	require.NoError(t, repo.Create(ctx, p))
	loaded, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Zero(t, loaded.DeliveryPlanVersion)
	require.JSONEq(t, `{"product_name":"","objective":"","success_metric":"","target_release":"","milestones":[],"test_cases":[]}`, string(loaded.DeliveryPlan))
	raw := json.RawMessage(`{"product_name":"Product","objective":"Ship","success_metric":"","target_release":"","milestones":[],"test_cases":[]}`)
	imported := &domain.Project{ID: uuid.New(), WorkspaceID: ws, Name: "Imported plan", Status: domain.ProjectStatusPlanned, DeliveryPlan: raw, DeliveryPlanVersion: 8}
	require.NoError(t, repo.Create(ctx, imported))
	created, err := repo.GetByID(ctx, imported.ID)
	require.NoError(t, err)
	require.Equal(t, 8, created.DeliveryPlanVersion)
	require.JSONEq(t, string(raw), string(created.DeliveryPlan))
	updated, err := repo.UpdateDeliveryPlan(ctx, uuid.New(), id, raw, 0)
	require.NoError(t, err)
	require.False(t, updated)
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.UpdateDeliveryPlan(ctx, ws, id, raw, 0)
			results <- ok
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	successes := 0
	for ok := range results {
		if ok {
			successes++
		}
	}
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, 1, successes)
	// Updating normal project metadata must not overwrite a separately saved plan.
	p.Name = "Renamed"
	require.NoError(t, repo.Update(ctx, p))
	loaded, err = repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 1, loaded.DeliveryPlanVersion)
	require.JSONEq(t, string(raw), string(loaded.DeliveryPlan))
}
