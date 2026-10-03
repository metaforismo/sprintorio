CREATE TABLE agent_tokens (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    prefix VARCHAR(16) NOT NULL,
    token_hash CHAR(64) UNIQUE NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('read', 'write')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX agent_tokens_owner_idx ON agent_tokens(workspace_id,user_id,created_at DESC);
