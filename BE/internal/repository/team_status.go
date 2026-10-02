package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
)

type TeamStatusRepository struct {
	db *sqlx.DB
}

func NewTeamStatusRepository(db *sqlx.DB) *TeamStatusRepository {
	return &TeamStatusRepository{db: db}
}

func (r *TeamStatusRepository) Create(ctx context.Context, status *domain.TeamStatus) error {
	query := `INSERT INTO team_statuses (id, team_id, name, slug, category, color, position, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		status.ID, status.TeamID, status.Name, status.Slug,
		status.Category, status.Color, status.Position, status.IsDefault,
	).Scan(&status.CreatedAt, &status.UpdatedAt)
}

func (r *TeamStatusRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TeamStatus, error) {
	var status domain.TeamStatus
	err := r.db.GetContext(ctx, &status, `SELECT * FROM team_statuses WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &status, err
}

func (r *TeamStatusRepository) GetByTeamAndSlug(ctx context.Context, teamID uuid.UUID, slug string) (*domain.TeamStatus, error) {
	var status domain.TeamStatus
	err := r.db.GetContext(ctx, &status, `SELECT * FROM team_statuses WHERE team_id = $1 AND slug = $2`, teamID, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &status, err
}

func (r *TeamStatusRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.TeamStatus, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM team_statuses WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var statuses []domain.TeamStatus
	err = r.db.SelectContext(ctx, &statuses, query, args...)
	return statuses, err
}

func (r *TeamStatusRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]domain.TeamStatus, error) {
	var statuses []domain.TeamStatus
	err := r.db.SelectContext(ctx, &statuses, `SELECT * FROM team_statuses WHERE team_id = $1 ORDER BY position`, teamID)
	return statuses, err
}

func (r *TeamStatusRepository) Update(ctx context.Context, status *domain.TeamStatus) error {
	query := `UPDATE team_statuses SET name = $1, color = $2, position = $3, updated_at = NOW()
		WHERE id = $4 RETURNING updated_at`
	return r.db.QueryRowContext(ctx, query, status.Name, status.Color, status.Position, status.ID).Scan(&status.UpdatedAt)
}

func (r *TeamStatusRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM team_statuses WHERE id = $1`, id)
	return err
}

func (r *TeamStatusRepository) NextPosition(ctx context.Context, teamID uuid.UUID) (int, error) {
	var pos int
	err := r.db.GetContext(ctx, &pos, `SELECT COALESCE(MAX(position), -1) + 1 FROM team_statuses WHERE team_id = $1`, teamID)
	return pos, err
}

var ErrStatusProjectScope = errors.New("projects must belong to the team's workspace")

func (r *TeamStatusRepository) CreateWithProjectVisibility(ctx context.Context, status *domain.TeamStatus, projectIDs []uuid.UUID) error {
	return r.persistWithProjectVisibility(ctx, status, &projectIDs, true)
}
func (r *TeamStatusRepository) UpdateWithProjectVisibility(ctx context.Context, status *domain.TeamStatus, projectIDs *[]uuid.UUID) error {
	return r.persistWithProjectVisibility(ctx, status, projectIDs, false)
}

// Updating only name/color/position preserves visibility. An explicitly supplied
// empty list clears it. Both metadata and visibility commit together.
func (r *TeamStatusRepository) persistWithProjectVisibility(ctx context.Context, status *domain.TeamStatus, projectIDs *[]uuid.UUID, create bool) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if projectIDs != nil && len(*projectIDs) > 0 {
		query, args, err := sqlx.In(`SELECT COUNT(*) FROM projects p JOIN teams t ON p.workspace_id=t.workspace_id WHERE t.id=? AND p.id IN (?)`, status.TeamID, *projectIDs)
		if err != nil {
			return err
		}
		var count int
		if err := tx.GetContext(ctx, &count, tx.Rebind(query), args...); err != nil {
			return err
		}
		if count != len(*projectIDs) {
			return ErrStatusProjectScope
		}
	}
	if create {
		err = tx.QueryRowContext(ctx, `INSERT INTO team_statuses(id,team_id,name,slug,category,color,position,is_default) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING created_at,updated_at`, status.ID, status.TeamID, status.Name, status.Slug, status.Category, status.Color, status.Position, status.IsDefault).Scan(&status.CreatedAt, &status.UpdatedAt)
	} else {
		err = tx.QueryRowContext(ctx, `UPDATE team_statuses SET name=$1,color=$2,position=$3,updated_at=NOW() WHERE id=$4 AND team_id=$5 RETURNING updated_at`, status.Name, status.Color, status.Position, status.ID, status.TeamID).Scan(&status.UpdatedAt)
	}
	if err != nil {
		return err
	}
	if projectIDs != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM project_status_visibility WHERE status_id=$1`, status.ID); err != nil {
			return err
		}
		for _, id := range *projectIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO project_status_visibility(project_id,status_id) VALUES($1,$2)`, id, status.ID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
