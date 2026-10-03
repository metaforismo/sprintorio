package domain

import (
	"time"

	"github.com/google/uuid"
)

// AgentToken deliberately never serializes its hash or ownership identifiers.
type AgentToken struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	WorkspaceID uuid.UUID  `db:"workspace_id" json:"-"`
	UserID      uuid.UUID  `db:"user_id" json:"-"`
	Name        string     `db:"name" json:"name"`
	Prefix      string     `db:"prefix" json:"prefix"`
	TokenHash   string     `db:"token_hash" json:"-"`
	Scope       string     `db:"scope" json:"scope"`
	ExpiresAt   time.Time  `db:"expires_at" json:"expires_at"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	LastUsedAt  *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	RevokedAt   *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
}
