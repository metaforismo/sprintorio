package migrations_test

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeliveryPlanMigrationRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	admin, err := sql.Open("pgx", databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })
	databaseName := "sprintorio_plan_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(`CREATE DATABASE "` + databaseName + `"`)
	if postgresCode(err) == "42501" {
		t.Skip("DATABASE_URL user cannot create isolated migration database")
	}
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = admin.Exec(`DROP DATABASE IF EXISTS "` + databaseName + `" WITH (FORCE)`) })
	testURL := databaseURLWithName(t, databaseURL, databaseName)
	migrator, err := migrate.New("file://.", testURL)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = migrator.Close() })
	require.NoError(t, migrator.Migrate(33))
	db, err := sql.Open("pgx", testURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	fixture := insertBaseMigrationFixture(t, db)
	require.NoError(t, migrator.Steps(1))
	requireMigrationVersion(t, migrator, 34)
	var defaultPlan string
	var version int
	require.NoError(t, db.QueryRow(`SELECT delivery_plan::text,delivery_plan_version FROM projects WHERE workspace_id=$1 LIMIT 1`, fixture.workspaceA).Scan(&defaultPlan, &version))
	require.Zero(t, version)
	require.JSONEq(t, `{"product_name":"","objective":"","success_metric":"","target_release":"","milestones":[],"test_cases":[]}`, defaultPlan)
	_, err = db.Exec(`UPDATE projects SET delivery_plan_version=-1 WHERE workspace_id=$1`, fixture.workspaceA)
	require.Equal(t, "23514", postgresCode(err))
	require.NoError(t, migrator.Steps(-1))
	requireMigrationVersion(t, migrator, 33)
	var columnCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_name='projects' AND column_name IN ('delivery_plan','delivery_plan_version')`).Scan(&columnCount))
	require.Zero(t, columnCount)
	var projectCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE workspace_id=$1`, fixture.workspaceA).Scan(&projectCount))
	require.Positive(t, projectCount)
}
