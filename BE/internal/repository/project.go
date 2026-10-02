package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/metaforismo/sprintorio/BE/internal/dto"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
)

type ProjectRepository struct {
	db *sqlx.DB
}

func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if len(project.DeliveryPlan) == 0 {
		plan := domain.DeliveryPlan{}
		plan.Normalize()
		project.DeliveryPlan, _ = json.Marshal(plan)
	}
	query := `INSERT INTO projects (id, workspace_id, team_id, name, description, status, lead_id, start_date, target_date, sort_order, delivery_plan, delivery_plan_version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12) RETURNING created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, project.ID, project.WorkspaceID, project.TeamID, project.Name, project.Description, project.Status, project.LeadID, project.StartDate, project.TargetDate, project.SortOrder, string(project.DeliveryPlan), project.DeliveryPlanVersion).Scan(&project.CreatedAt, &project.UpdatedAt)
}

func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	var project domain.Project
	err := r.db.GetContext(ctx, &project, `SELECT * FROM projects WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &project, err
}

func (r *ProjectRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Project, error) {
	var projects []domain.Project
	err := r.db.SelectContext(ctx, &projects, `SELECT * FROM projects WHERE workspace_id = $1 ORDER BY sort_order, name`, workspaceID)
	return projects, err
}

func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	query := `UPDATE projects SET name = $1, description = $2, status = $3, lead_id = $4, start_date = $5, target_date = $6, sort_order = $7, team_id = $8, updated_at = NOW() WHERE id = $9 RETURNING updated_at`
	return r.db.QueryRowContext(ctx, query, project.Name, project.Description, project.Status, project.LeadID, project.StartDate, project.TargetDate, project.SortOrder, project.TeamID, project.ID).Scan(&project.UpdatedAt)
}

func (r *ProjectRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]domain.Project, error) {
	var projects []domain.Project
	err := r.db.SelectContext(ctx, &projects, `SELECT * FROM projects WHERE team_id = $1 ORDER BY sort_order, name`, teamID)
	return projects, err
}

func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

func (r *ProjectRepository) IssueStats(ctx context.Context, projectID uuid.UUID) (total int, completed int, cancelled int, err error) {
	err = r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*), COUNT(*) FILTER (WHERE %s = 'completed'), COUNT(*) FILTER (WHERE %s = 'cancelled') FROM issues i LEFT JOIN team_statuses ts ON ts.id=i.status_id WHERE i.project_id = $1`, issueStatusCategoryExpr("i", "ts"), issueStatusCategoryExpr("i", "ts")),
		projectID,
	).Scan(&total, &completed, &cancelled)
	return
}

// UpdateDeliveryPlan uses a single compare-and-swap statement so concurrent saves
// cannot overwrite each other, even after both callers read the same version.
func (r *ProjectRepository) UpdateDeliveryPlan(ctx context.Context, workspaceID, projectID uuid.UUID, plan json.RawMessage, version int) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE projects SET delivery_plan=$1::jsonb, delivery_plan_version=delivery_plan_version+1, updated_at=NOW() WHERE id=$2 AND workspace_id=$3 AND delivery_plan_version=$4`, string(plan), projectID, workspaceID, version)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (r *ProjectRepository) ValidateReferences(ctx context.Context, workspaceID uuid.UUID, teamID, leadID *uuid.UUID) (bool, error) {
	var valid bool
	err := r.db.QueryRowContext(ctx, `SELECT ($2::uuid IS NULL OR EXISTS(SELECT 1 FROM teams WHERE id=$2 AND workspace_id=$1)) AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM workspace_members WHERE user_id=$3 AND workspace_id=$1))`, workspaceID, teamID, leadID).Scan(&valid)
	return valid, err
}
func (r *ProjectRepository) IssueStatsByWorkspace(ctx context.Context, workspaceID uuid.UUID) (map[uuid.UUID]dto.ProjectProgressResponse, error) {
	type row struct {
		ID        uuid.UUID `db:"project_id"`
		Total     int       `db:"total"`
		Completed int       `db:"completed"`
		Cancelled int       `db:"cancelled"`
	}
	var rows []row
	category := issueStatusCategoryExpr("i", "ts")
	err := r.db.SelectContext(ctx, &rows, fmt.Sprintf(`SELECT p.id project_id,COUNT(i.id) total,COUNT(i.id) FILTER (WHERE %s='completed') completed,COUNT(i.id) FILTER (WHERE %s='cancelled') cancelled FROM projects p LEFT JOIN issues i ON i.project_id=p.id AND i.workspace_id=p.workspace_id LEFT JOIN team_statuses ts ON ts.id=i.status_id WHERE p.workspace_id=$1 GROUP BY p.id`, category, category), workspaceID)
	result := make(map[uuid.UUID]dto.ProjectProgressResponse, len(rows))
	for _, r := range rows {
		result[r.ID] = dto.ProjectProgressResponse{Total: r.Total, Completed: r.Completed, Cancelled: r.Cancelled}
	}
	return result, err
}
