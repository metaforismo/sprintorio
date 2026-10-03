ALTER TABLE agent_tokens DROP CONSTRAINT agent_tokens_scope_check;
ALTER TABLE agent_tokens ADD CONSTRAINT agent_tokens_scope_check CHECK (scope IN ('read','write','full'));
