import { api } from './client';
import type { AgentToken, AgentScope, CreatedAgentToken } from '$lib/types/agent-token';
const path = (slug: string) => `/api/workspaces/${encodeURIComponent(slug)}/agent-tokens`;
export const listAgentTokens = (slug: string) => api.get<AgentToken[]>(path(slug));
export const createAgentToken = (slug: string, data: { name: string; scope: AgentScope; expires_in_days: number }) =>
	api.post<CreatedAgentToken>(path(slug), data);
export const revokeAgentToken = (slug: string, id: string) =>
	api.delete<void>(`${path(slug)}/${encodeURIComponent(id)}`);
