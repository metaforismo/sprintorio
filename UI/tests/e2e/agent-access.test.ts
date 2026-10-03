import { expect, test, type Page } from '@playwright/test';

const workspace = {
	id: 'workspace',
	name: 'Agent workspace',
	slug: 'test',
	current_user_role: 'owner',
	owner_id: 'user'
};
const record = {
	id: 'token-1',
	name: 'Codex',
	prefix: 'spr_test',
	scope: 'read',
	expires_at: '2030-01-01T00:00:00Z',
	created_at: '2026-01-01T00:00:00Z'
};
const secret = 'spr_test_secret_only_for_mocked_test';
async function setup(page: Page, handler: (route: import('@playwright/test').Route) => Promise<void>) {
	await page.route('https://raw.githubusercontent.com/**', (route) => route.fulfill({ json: [] }));
	await page.route('**://*/api/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		if (path.startsWith('/api/workspaces/test/agent-tokens')) return handler(route);
		if (path === '/api/auth/me')
			return route.fulfill({ json: { id: 'user', name: 'Ada', email: 'ada@example.test', is_sysadmin: false } });
		if (path === '/api/preferences') return route.fulfill({ json: {} });
		if (path === '/api/workspaces') return route.fulfill({ json: [workspace] });
		if (path === '/api/workspaces/test') return route.fulfill({ json: workspace });
		if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
		return route.fulfill({ json: [] });
	});
}
// These tests deliberately render a dummy secret. Disable all captured artifacts.
test.use({ trace: 'off', screenshot: 'off', video: 'off' });

test('retry, safe defaults, failed draft, one-time secret and placeholder configuration', async ({ page, context }) => {
	let lists = 0,
		creates = 0;
	let requestBody: any;
	await context.grantPermissions(['clipboard-read', 'clipboard-write']);
	await setup(page, async (route) => {
		if (route.request().method() === 'GET')
			return ++lists === 1 ? route.fulfill({ status: 503, json: {} }) : route.fulfill({ json: [] });
		requestBody = route.request().postDataJSON();
		if (++creates === 1) return route.fulfill({ status: 503, json: {} });
		return route.fulfill({ status: 201, json: { token: secret, record } });
	});
	await page.goto('/test/settings/agents');
	await expect(page.getByRole('alert')).toContainText('Could not load tokens');
	await page.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(page.getByText('Connect your first agent')).toBeVisible();
	await page.getByRole('button', { name: 'Create token', exact: true }).click();
	let dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('radio', { name: /Read only/ })).toBeChecked();
	await expect(dialog.getByLabel('Expires in days')).toHaveValue('30');
	await dialog.getByLabel('Name', { exact: true }).fill('Codex');
	await dialog.getByRole('button', { name: 'Create token', exact: true }).click();
	await expect(dialog.getByRole('alert')).toContainText('Could not create token');
	await expect(dialog.getByLabel('Name', { exact: true })).toHaveValue('Codex');
	await dialog.getByRole('button', { name: 'Create token', exact: true }).click();
	await expect(dialog.getByRole('heading', { name: 'Save your token' })).toBeVisible();
	expect(requestBody).toEqual({ name: 'Codex', scope: 'read', expires_in_days: 30 });
	await expect(dialog.locator('[data-agent-secret]')).toHaveText(secret);
	await dialog.getByRole('button', { name: 'Copy token', exact: true }).click();
	expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(secret);
	expect(
		await page.evaluate(
			(value) =>
				JSON.stringify(localStorage).includes(value) ||
				JSON.stringify(sessionStorage).includes(value) ||
				location.href.includes(value),
			secret
		)
	).toBe(false);
	await dialog.getByRole('button', { name: 'Done', exact: true }).click();
	await expect(page.getByText(secret, { exact: true })).toHaveCount(0);
	await page.getByRole('button', { name: 'Copy configuration' }).click();
	const config = JSON.parse(await page.evaluate(() => navigator.clipboard.readText()));
	expect(config.mcpServers.sprintorio.env).toEqual({
		SPRINTORIO_URL: 'http://localhost:4174',
		SPRINTORIO_TOKEN: 'YOUR_TOKEN',
		SPRINTORIO_WORKSPACE: 'test'
	});
	await page.getByLabel('Agent client').selectOption('codex');
	await page.getByRole('button', { name: 'Copy configuration' }).click();
	const codexConfig = await page.evaluate(() => navigator.clipboard.readText());
	expect(codexConfig).toContain('[mcp_servers.sprintorio]');
	expect(codexConfig).toContain('env_vars = ["SPRINTORIO_TOKEN"]');
	expect(codexConfig).not.toContain(secret);
	await page.getByLabel('Agent client').selectOption('muse');
	await page.getByRole('button', { name: 'Copy configuration' }).click();
	const museConfig = JSON.parse(await page.evaluate(() => navigator.clipboard.readText()));
	expect(museConfig.mcp_servers.sprintorio.transport).toBe('stdio');
	expect(museConfig.mcp_servers.sprintorio.env.SPRINTORIO_TOKEN).toBe('YOUR_TOKEN');
	expect(museConfig.mcpServers).toBeUndefined();
	await page.reload();
	await expect(page.getByText(secret, { exact: true })).toHaveCount(0);
});

test('write opt-in, pending protection, revoke failure and confirmation', async ({ page }) => {
	let created = false,
		deletes = 0;
	let start = false;
	let release!: () => void;
	const pending = new Promise<void>((resolve) => (release = resolve));
	await setup(page, async (route) => {
		if (route.request().method() === 'GET')
			return route.fulfill({ json: created ? [{ ...record, scope: 'write' }] : [] });
		if (route.request().method() === 'DELETE')
			return ++deletes === 1 ? route.fulfill({ status: 503, json: {} }) : route.fulfill({ status: 204 });
		expect(route.request().postDataJSON()).toEqual({ name: 'Codex', scope: 'write', expires_in_days: 7 });
		start = true;
		await pending;
		created = true;
		return route.fulfill({ status: 201, json: { token: secret, record: { ...record, scope: 'write' } } });
	});
	await page.goto('/test/settings/agents');
	await page.getByRole('button', { name: 'Create token', exact: true }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByLabel('Name', { exact: true }).fill('Codex');
	await dialog.getByRole('radio', { name: /Read and write/ }).check();
	await dialog.getByLabel('Expires in days').fill('7');
	try {
		await dialog.getByRole('button', { name: 'Create token', exact: true }).click();
		await expect.poll(() => start).toBe(true);
		await page.keyboard.press('Escape');
		await expect(dialog).toBeVisible();
		await expect(dialog.locator('form')).toHaveAttribute('aria-busy', 'true');
		await expect(dialog.getByRole('button', { name: 'Creating…', exact: true })).toBeDisabled();
	} finally {
		release();
	}
	await expect(dialog.getByRole('heading', { name: 'Save your token' })).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(page.locator('[data-agent-secret]')).toHaveCount(0);
	await page.getByRole('button', { name: 'Revoke Codex' }).click();
	await expect(dialog.getByRole('heading', { name: 'Revoke access?' })).toBeVisible();
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	expect(deletes).toBe(0);
	await page.getByRole('button', { name: 'Revoke Codex' }).click();
	await dialog.getByRole('button', { name: 'Revoke', exact: true }).click();
	await expect(dialog.getByRole('alert')).toContainText('Could not revoke token');
	await dialog.getByRole('button', { name: 'Revoke', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await expect(page.getByText('No active tokens', { exact: true })).toBeVisible();
	await page.locator('summary').filter({ hasText: 'Expired and revoked (1)' }).click();
	await expect(page.getByText('Revoked', { exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Revoke Codex' })).toHaveCount(0);
});

test('settings remains usable on mobile and secret clears on navigation', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await setup(page, async (route) =>
		route.request().method() === 'POST'
			? route.fulfill({ status: 201, json: { token: secret, record } })
			: route.fulfill({ json: [] })
	);
	await page.goto('/test/settings/agents');
	await page.getByRole('button', { name: 'Create token', exact: true }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByLabel('Name', { exact: true }).fill('Codex');
	await dialog.getByRole('button', { name: 'Create token', exact: true }).click();
	await expect(dialog.locator('[data-agent-secret]')).toHaveText(secret);
	await page.goto('/test/settings/profile');
	await expect(page.getByText(secret, { exact: true })).toHaveCount(0);
	await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});

test('full workspace access is explicit and submitted with the current role boundary', async ({ page }) => {
	let requestBody: any;
	await setup(page, async (route) => {
		if (route.request().method() === 'GET') return route.fulfill({ json: [] });
		requestBody = route.request().postDataJSON();
		return route.fulfill({ status: 201, json: { token: secret, record: { ...record, scope: 'full' } } });
	});
	await page.goto('/test/settings/agents');
	await page.getByRole('button', { name: 'Create token', exact: true }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('radio', { name: /Read only/ })).toBeChecked();
	await dialog.getByLabel('Name', { exact: true }).fill('Codex');
	await dialog.getByRole('radio', { name: /Full workspace access/ }).check();
	await expect(
		dialog.getByText('All workspace actions allowed by your current role. Token management stays in the app.')
	).toBeVisible();
	await dialog.getByRole('button', { name: 'Create token', exact: true }).click();
	await expect(dialog.getByRole('heading', { name: 'Save your token' })).toBeVisible();
	expect(requestBody).toEqual({ name: 'Codex', scope: 'full', expires_in_days: 30 });
	await dialog.getByRole('button', { name: 'Done', exact: true }).click();
	await expect(
		page.getByRole('region', { name: 'Agent access', exact: true }).getByText(/Full workspace access/)
	).toBeVisible();
});

test('inactive history stays collapsed while active access and connection setup remain available', async ({ page }) => {
	const history = Array.from({ length: 7 }, (_, index) => ({
		...record,
		id: `old-${index}`,
		name: `Previous agent ${index}`,
		revoked_at: '2026-01-02T00:00:00Z'
	}));
	const expired = { ...record, id: 'expired', name: 'Expired agent', expires_at: '2020-01-01T00:00:00Z' };
	await setup(page, (route) =>
		route.request().method() === 'DELETE'
			? route.fulfill({ status: 204 })
			: route.fulfill({ json: [...history, expired, record] })
	);
	await page.goto('/test/settings/agents');
	const disclosure = page
		.locator('details')
		.filter({ has: page.locator('summary').filter({ hasText: 'Expired and revoked' }) });
	await expect(disclosure.locator('summary')).toHaveText('Expired and revoked (8)');
	await expect(disclosure).toHaveJSProperty('open', false);
	await expect(page.getByRole('heading', { name: 'Previous agent 0', exact: true })).not.toBeVisible();
	await expect(page.getByRole('button', { name: 'Revoke Codex' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Connect an agent', exact: true })).toBeVisible();
	await disclosure.locator('summary').click();
	await expect(page.getByRole('heading', { name: 'Previous agent 0', exact: true })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Expired agent', exact: true })).toBeVisible();
	await expect(page.getByText('Expired', { exact: true })).toBeVisible();
	await disclosure.locator('summary').click();
	await page.getByRole('button', { name: 'Revoke Codex' }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Revoke', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'No active tokens', exact: true })).toBeVisible();
	await expect(disclosure.locator('summary')).toHaveText('Expired and revoked (9)');
	await expect(page.getByRole('heading', { name: 'Codex', exact: true })).not.toBeVisible();
	await expect(page.getByRole('button', { name: 'Revoke Codex' })).toHaveCount(0);
	await disclosure.locator('summary').click();
	await expect(page.getByRole('heading', { name: 'Codex', exact: true })).toBeVisible();
});
