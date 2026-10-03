export type AgentScope = 'read' | 'write' | 'full';
export interface AgentToken {
	id: string;
	name: string;
	prefix: string;
	scope: AgentScope;
	expires_at: string;
	created_at: string;
	last_used_at?: string | null;
	revoked_at?: string | null;
}
export interface CreatedAgentToken {
	token: string;
	record: AgentToken;
}
