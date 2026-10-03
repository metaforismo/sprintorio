import assert from 'node:assert/strict';
import test from 'node:test';
import { agentConfiguration, agentClients } from '../../src/lib/features/workspaces/agent-config.ts';

test('Codex uses TOML and forwards the token from the environment', () => {
	const config = agentConfiguration('codex', 'https://sprintorio.example', 'my-workspace');
	assert.match(config, /^\[mcp_servers\.sprintorio\]/);
	assert.match(config, /env_vars = \["SPRINTORIO_TOKEN"\]/);
	assert.match(config, /SPRINTORIO_URL = "https:\/\/sprintorio.example"/);
	assert.match(config, /SPRINTORIO_WORKSPACE = "my-workspace"/);
	assert.equal(config.includes('YOUR_TOKEN'), false);
	assert.equal(config.includes('SPRINTORIO_TOKEN ='), false);
});
test('Muse, Claude, Gemini and OpenCode keep their distinct schema and placeholder secret', () => {
	const muse = JSON.parse(agentConfiguration('muse', 'https://server.example', 'team'));
	assert.equal(muse.mcp_servers.sprintorio.transport, 'stdio');
	assert.equal(muse.mcpServers, undefined);
	const claude = JSON.parse(agentConfiguration('claude', 'https://server.example', 'team'));
	assert.equal(claude.mcpServers.sprintorio.type, 'stdio');
	const gemini = JSON.parse(agentConfiguration('gemini', 'https://server.example', 'team'));
	assert.equal(gemini.mcpServers.sprintorio.type, undefined);
	const open = JSON.parse(agentConfiguration('opencode', 'https://server.example', 'team'));
	assert.deepEqual(open.mcp.sprintorio.command, ['sprintorio', 'mcp']);
	for (const client of agentClients.filter((value) => value.id !== 'codex')) {
		const config = agentConfiguration(client.id, 'https://server.example', 'team');
		assert.equal(config.includes('YOUR_TOKEN'), true);
	}
});
