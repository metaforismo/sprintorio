import assert from 'node:assert/strict';
import test from 'node:test';
import { preferredWorkspace } from '../../src/lib/utils/workspace-navigation.ts';
import { createShortcutEngine } from '../../src/lib/utils/keyboard.ts';

test('restores a remembered workspace only when it is accessible', () => {
	const workspaces = [{ slug: 'first' }, { slug: 'recent' }];
	assert.equal(preferredWorkspace(workspaces, 'recent'), workspaces[1]);
	assert.equal(preferredWorkspace(workspaces, 'removed'), workspaces[0]);
	assert.equal(preferredWorkspace([], 'recent'), undefined);
});

test('keyboard engine reads current workspace actions without rebuilding', () => {
	let workspace = 'first';
	let visited = '';
	const engine = createShortcutEngine(() => [
		{
			keys: ['g', 'i'],
			handler: () => {
				visited = workspace;
			},
			label: 'Inbox',
			category: 'Navigation'
		}
	]);
	const press = (key) => engine.handler({ key, target: { tagName: 'BODY' }, preventDefault() {} });
	press('g');
	press('i');
	assert.equal(visited, 'first');
	workspace = 'recent';
	press('g');
	press('i');
	assert.equal(visited, 'recent');
});
