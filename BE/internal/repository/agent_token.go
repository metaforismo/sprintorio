package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
)

type AgentTokenRepository struct{ db *sqlx.DB }

func NewAgentTokenRepository(db *sqlx.DB) *AgentTokenRepository { return &AgentTokenRepository{db: db} }
func (r *AgentTokenRepository) Create(ctx context.Context, token *domain.AgentToken) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO agent_tokens(id,workspace_id,user_id,name,prefix,token_hash,scope,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING created_at`, token.ID, token.WorkspaceID, token.UserID, token.Name, token.Prefix, token.TokenHash, token.Scope, token.ExpiresAt).Scan(&token.CreatedAt)
}
func (r *AgentTokenRepository) List(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.AgentToken, error) {
	tokens := make([]domain.AgentToken, 0)
	err := r.db.SelectContext(ctx, &tokens, `SELECT * FROM agent_tokens WHERE workspace_id=$1 AND user_id=$2 ORDER BY created_at DESC`, workspaceID, userID)
	return tokens, err
}
func (r *AgentTokenRepository) Revoke(ctx context.Context, id, workspaceID, userID uuid.UUID) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE agent_tokens SET revoked_at=COALESCE(revoked_at,NOW()) WHERE id=$1 AND workspace_id=$2 AND user_id=$3`, id, workspaceID, userID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

// Each authentication reads the authoritative row, so revocation is never cached.
func (r *AgentTokenRepository) FindActive(ctx context.Context, hash string) (*domain.AgentToken, error) {
	var token domain.AgentToken
	err := r.db.GetContext(ctx, &token, `SELECT t.* FROM agent_tokens t JOIN workspace_members m ON m.workspace_id=t.workspace_id AND m.user_id=t.user_id WHERE t.token_hash=$1 AND t.revoked_at IS NULL AND t.expires_at>NOW()`, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &token, err
}
func (r *AgentTokenRepository) Touch(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE agent_tokens SET last_used_at=NOW() WHERE id=$1 AND (last_used_at IS NULL OR last_used_at<NOW()-INTERVAL '15 minutes')`, id)
	return err
}
