export type AgentClient = 'claude' | 'cursor' | 'codex' | 'muse' | 'gemini' | 'opencode';
export const agentClients: { id: AgentClient; name: string }[] = [
	{ id: 'claude', name: 'Claude Code' },
	{ id: 'cursor', name: 'Cursor' },
	{ id: 'codex', name: 'Codex' },
	{ id: 'muse', name: 'Muse Code' },
	{ id: 'gemini', name: 'Gemini CLI' },
	{ id: 'opencode', name: 'OpenCode' }
];
// Secrets never enter this helper. Templates always use a placeholder or inherited environment.
export function agentConfiguration(client: AgentClient, url: string, workspace: string): string {
	const env = { SPRINTORIO_URL: url, SPRINTORIO_TOKEN: 'YOUR_TOKEN', SPRINTORIO_WORKSPACE: workspace };
	if (client === 'codex')
		return `[mcp_servers.sprintorio]\ncommand = "sprintorio"\nargs = ["mcp"]\nenv_vars = ["SPRINTORIO_TOKEN"]\n\n[mcp_servers.sprintorio.env]\nSPRINTORIO_URL = ${JSON.stringify(url)}\nSPRINTORIO_WORKSPACE = ${JSON.stringify(workspace)}`;
	if (client === 'muse')
		return JSON.stringify(
			{ mcp_servers: { sprintorio: { transport: 'stdio', command: 'sprintorio', args: ['mcp'], env } } },
			null,
			2
		);
	if (client === 'opencode')
		return JSON.stringify(
			{ mcp: { sprintorio: { type: 'local', command: ['sprintorio', 'mcp'], environment: env } } },
			null,
			2
		);
	return JSON.stringify(
		{
			mcpServers: {
				sprintorio: { ...(client === 'gemini' ? {} : { type: 'stdio' }), command: 'sprintorio', args: ['mcp'], env }
			}
		},
		null,
		2
	);
}
