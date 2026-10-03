package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// WorkspaceResourceRepository checks identifiers before any read or mutation.
// Only compile-time table names may enter the query; request data stays parameterized.
type WorkspaceResourceRepository struct{ db *sqlx.DB }

func NewWorkspaceResourceRepository(db *sqlx.DB) *WorkspaceResourceRepository {
	return &WorkspaceResourceRepository{db: db}
}
func (r *WorkspaceResourceRepository) Accessible(ctx context.Context, resource string, id, workspaceID, userID uuid.UUID, write bool) (bool, error) {
	tables := map[string]string{"views": "views", "issue-templates": "issue_templates", "shared-links": "shared_links", "webhooks": "webhooks", "labels": "labels", "projects": "projects", "favorites": "favorites", "github/repos": "github_repos"}
	table, ok := tables[resource]
	if !ok {
		return false, fmt.Errorf("unknown workspace resource")
	}
	query := `SELECT EXISTS(SELECT 1 FROM ` + table + ` WHERE id=$1 AND workspace_id=$2`
	args := []any{id, workspaceID}
	if resource == "views" {
		query += ` AND (creator_id=$3 OR (is_shared=TRUE AND $4=FALSE))`
		args = append(args, userID, write)
	}
	if resource == "favorites" {
		query += ` AND user_id=$3`
		args = append(args, userID)
	}
	query += `)`
	var allowed bool
	err := r.db.GetContext(ctx, &allowed, query, args...)
	return allowed, err
}
