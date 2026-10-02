package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, team *domain.Team) error {
	query := `INSERT INTO teams (id, workspace_id, name, key, description, color, icon) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, team.ID, team.WorkspaceID, team.Name, team.Key, team.Description, team.Color, team.Icon).Scan(&team.CreatedAt, &team.UpdatedAt)
}

func (r *TeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Team, error) {
	var team domain.Team
	err := r.db.GetContext(ctx, &team, `SELECT * FROM teams WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &team, err
}

func (r *TeamRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Team, error) {
	var teams []domain.Team
	err := r.db.SelectContext(ctx, &teams, `SELECT * FROM teams WHERE workspace_id = $1 ORDER BY name`, workspaceID)
	return teams, err
}

func (r *TeamRepository) Update(ctx context.Context, team *domain.Team) error {
	query := `UPDATE teams SET name = $1, description = $2, color = $3, icon = $4, triage_enabled = $5, parent_auto_close_enabled = $6, sub_issue_auto_close_enabled = $7, issue_copy_prompt = $8, updated_at = NOW() WHERE id = $9 RETURNING updated_at`
	return r.db.QueryRowContext(ctx, query, team.Name, team.Description, team.Color, team.Icon, team.TriageEnabled, team.ParentAutoCloseEnabled, team.SubIssueAutoCloseEnabled, team.IssueCopyPrompt, team.ID).Scan(&team.UpdatedAt)
}

func (r *TeamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}

func (r *TeamRepository) AddMember(ctx context.Context, member *domain.TeamMember) error {
	query := `INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING created_at`
	return r.db.QueryRowContext(ctx, query, member.TeamID, member.UserID).Scan(&member.CreatedAt)
}

func (r *TeamRepository) GetMember(ctx context.Context, teamID, userID uuid.UUID) (*domain.TeamMember, error) {
	var member domain.TeamMember
	err := r.db.GetContext(ctx, &member, `SELECT * FROM team_members WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &member, err
}

func (r *TeamRepository) ListMembers(ctx context.Context, teamID uuid.UUID) ([]domain.TeamMember, error) {
	var members []domain.TeamMember
	err := r.db.SelectContext(ctx, &members, `SELECT * FROM team_members WHERE team_id = $1`, teamID)
	return members, err
}

func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	return err
}

var ErrTeamKeyTaken = errors.New("team key is already used in this workspace")

func (r *TeamRepository) CreateWithMemberAndStatuses(ctx context.Context, team *domain.Team, member *domain.TeamMember, statuses []domain.TeamStatus) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `INSERT INTO teams(id,workspace_id,name,key,description,color,icon) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING created_at,updated_at`, team.ID, team.WorkspaceID, team.Name, team.Key, team.Description, team.Color, team.Icon).Scan(&team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "teams_workspace_id_key_key" {
			return ErrTeamKeyTaken
		}
		return err
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO team_members(team_id,user_id) VALUES($1,$2) RETURNING created_at`, member.TeamID, member.UserID).Scan(&member.CreatedAt)
	if err != nil {
		return err
	}
	for i := range statuses {
		s := &statuses[i]
		err = tx.QueryRowContext(ctx, `INSERT INTO team_statuses(id,team_id,name,slug,category,position,is_default) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING created_at,updated_at`, s.ID, s.TeamID, s.Name, s.Slug, s.Category, s.Position, s.IsDefault).Scan(&s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
