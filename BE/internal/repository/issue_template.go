package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
)

type IssueTemplateRepository struct {
	db *sqlx.DB
}

func NewIssueTemplateRepository(db *sqlx.DB) *IssueTemplateRepository {
	return &IssueTemplateRepository{db: db}
}

func (r *IssueTemplateRepository) Create(ctx context.Context, tmpl *domain.IssueTemplate) error {
	if err := r.validateReferences(ctx, tmpl); err != nil {
		return err
	}
	query := `INSERT INTO issue_templates (id, workspace_id, team_id, title, description, status, priority, assignee_id, label_ids, recurrence_rule, next_run_at, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		tmpl.ID, tmpl.WorkspaceID, tmpl.TeamID, tmpl.Title, tmpl.Description,
		tmpl.Status, tmpl.Priority, tmpl.AssigneeID, tmpl.LabelIDs,
		tmpl.RecurrenceRule, tmpl.NextRunAt, tmpl.IsActive, tmpl.CreatedBy,
	).Scan(&tmpl.CreatedAt, &tmpl.UpdatedAt)
}

func (r *IssueTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.IssueTemplate, error) {
	var tmpl domain.IssueTemplate
	err := r.db.GetContext(ctx, &tmpl, `SELECT * FROM issue_templates WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &tmpl, err
}

func (r *IssueTemplateRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.IssueTemplate, error) {
	var templates []domain.IssueTemplate
	err := r.db.SelectContext(ctx, &templates, `SELECT * FROM issue_templates WHERE workspace_id = $1 ORDER BY title`, workspaceID)
	return templates, err
}

func (r *IssueTemplateRepository) Update(ctx context.Context, tmpl *domain.IssueTemplate) error {
	if err := r.validateReferences(ctx, tmpl); err != nil {
		return err
	}
	query := `UPDATE issue_templates SET
		title = $1, description = $2, status = $3, priority = $4, assignee_id = $5,
		team_id = $6, label_ids = $7, recurrence_rule = $8, next_run_at = $9,
		is_active = $10, updated_at = NOW()
		WHERE id = $11 RETURNING updated_at`
	return r.db.QueryRowContext(ctx, query,
		tmpl.Title, tmpl.Description, tmpl.Status, tmpl.Priority, tmpl.AssigneeID,
		tmpl.TeamID, tmpl.LabelIDs, tmpl.RecurrenceRule, tmpl.NextRunAt,
		tmpl.IsActive, tmpl.ID,
	).Scan(&tmpl.UpdatedAt)
}

func (r *IssueTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM issue_templates WHERE id = $1`, id)
	return err
}

func (r *IssueTemplateRepository) ListDueForRecurrence(ctx context.Context) ([]domain.IssueTemplate, error) {
	var templates []domain.IssueTemplate
	err := r.db.SelectContext(ctx, &templates, `SELECT * FROM issue_templates WHERE is_active = true AND next_run_at IS NOT NULL AND next_run_at <= NOW()`)
	return templates, err
}

// Templates must not reference another workspace's team, members or labels.
func (r *IssueTemplateRepository) validateReferences(ctx context.Context, tmpl *domain.IssueTemplate) error {
	labels := string(tmpl.LabelIDs)
	if labels == "" {
		labels = "[]"
	}
	var valid bool
	err := r.db.GetContext(ctx, &valid, `SELECT
 ($2::uuid IS NULL OR EXISTS(SELECT 1 FROM teams WHERE id=$2 AND workspace_id=$1))
 AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM workspace_members WHERE user_id=$3 AND workspace_id=$1))
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements_text($4::jsonb) AS requested(id)
 WHERE NOT EXISTS(SELECT 1 FROM labels WHERE labels.id::text=LOWER(requested.id) AND workspace_id=$1))`, tmpl.WorkspaceID, tmpl.TeamID, tmpl.AssigneeID, labels)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("template references must belong to its workspace")
	}
	return nil
}
