import { expect, test } from '@playwright/test';

const user = { id: 'user', name: 'Ada', email: 'ada@example.test', is_sysadmin: false };
const workspace = {
	id: 'workspace',
	name: 'Ada workspace',
	slug: 'test',
	current_user_role: 'owner',
	owner_id: 'user'
};
const team = { id: 'team', name: 'Engineering', key: 'ENG', triage_enabled: false };

test('first issue continues after team creation and preserves both failed drafts', async ({ page }) => {
	let teamAttempts = 0;
	let issueAttempts = 0;
	let createdTeam = false;
	let firstTeamRequestStarted = false;
	let releaseFirstTeamRequest!: () => void;
	const firstTeamRequest = new Promise<void>((resolve) => {
		releaseFirstTeamRequest = resolve;
	});
	await page.route('https://raw.githubusercontent.com/**', (route) => route.fulfill({ json: [] }));
	await page.route('**://*/api/**', async (route) => {
		const request = route.request();
		const path = new URL(request.url()).pathname;
		if (path === '/api/auth/me') return route.fulfill({ json: user });
		if (path === '/api/preferences') return route.fulfill({ json: {} });
		if (path === '/api/workspaces') return route.fulfill({ json: [workspace] });
		if (path === '/api/workspaces/test') return route.fulfill({ json: workspace });
		if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
		if (path === '/api/workspaces/test/teams' && request.method() === 'POST') {
			teamAttempts++;
			if (teamAttempts === 1) {
				firstTeamRequestStarted = true;
				await firstTeamRequest;
				return route.fulfill({ status: 409, json: { error: { message: 'Prefix already used' } } });
			}
			createdTeam = true;
			return route.fulfill({ status: 201, json: team });
		}
		if (path === '/api/workspaces/test/teams') return route.fulfill({ json: createdTeam ? [team] : [] });
		if (path === '/api/workspaces/test/issues' && request.method() === 'POST') {
			issueAttempts++;
			if (issueAttempts === 1) return route.fulfill({ status: 503, json: { error: { message: 'Please retry' } } });
			return route.fulfill({
				status: 201,
				json: { ...request.postDataJSON(), id: 'issue', identifier: 'ENG-1', status: 'backlog' }
			});
		}
		if (path === '/api/workspaces/test/issues')
			return route.fulfill({ json: { data: [], total_count: 0, has_more: false } });
		if (/\/statuses$/.test(path)) return route.fulfill({ json: [] });
		return route.fulfill({ json: [] });
	});
	await page.goto('/test/inbox');
	const guide = page.getByRole('region', { name: 'Start your workspace' });
	await expect(guide).toBeVisible();
	await page.getByRole('button', { name: 'New issue', exact: true }).click();
	const teamDialog = page.getByRole('dialog');
	await expect(teamDialog.getByRole('heading', { name: 'Create team' })).toBeVisible();
	await teamDialog.getByLabel('Name', { exact: true }).fill('Engineering');
	try {
		await teamDialog.getByRole('button', { name: 'Create team', exact: true }).click();
		await expect.poll(() => firstTeamRequestStarted).toBe(true);
		await page.keyboard.press('Escape');
		await expect(teamDialog).toBeVisible();
		await expect(teamDialog.locator('form')).toHaveAttribute('aria-busy', 'true');
		await expect(teamDialog.locator('button[type=submit]')).toBeDisabled();
		await expect(teamDialog.locator('button[type=submit]')).toHaveText('Creating...');
		await expect(teamDialog.getByLabel('Name', { exact: true })).toHaveValue('Engineering');
	} finally {
		releaseFirstTeamRequest();
	}
	await expect(page.locator('.app-toast-shell')).toContainText('Prefix already used');
	await expect(teamDialog.getByLabel('Name', { exact: true })).toHaveValue('Engineering');
	await teamDialog.getByRole('button', { name: 'Create team', exact: true }).click();
	const issueDialog = page.getByRole('dialog');
	await expect(issueDialog.getByPlaceholder('Issue title')).toBeVisible();
	// The shared selector is a single focusable trigger and supports keyboard selection.
	const priorityTrigger = issueDialog.getByRole('button', { name: 'No priority', exact: true });
	await priorityTrigger.focus();
	await page.keyboard.press('Enter');
	await expect(page.getByRole('option', { name: 'High', exact: true })).toBeVisible();
	await page.keyboard.press('Home');
	await page.keyboard.press('ArrowDown');
	await page.keyboard.press('ArrowDown');
	await page.keyboard.press('Enter');
	await expect(issueDialog.getByRole('button', { name: 'High', exact: true })).toBeFocused();

	await issueDialog.getByPlaceholder('Issue title').fill('Ship the first version');
	await issueDialog.getByRole('button', { name: 'Create issue', exact: true }).click();
	await expect(page.locator('.app-toast-shell').filter({ hasText: 'Please retry' })).toBeVisible();
	await expect(issueDialog.getByPlaceholder('Issue title')).toHaveValue('Ship the first version');
	await issueDialog.getByRole('button', { name: 'Create issue', exact: true }).click();
	await expect(issueDialog).not.toBeVisible();
	await expect(guide).not.toBeVisible();
	expect(teamAttempts).toBe(2);
	expect(issueAttempts).toBe(2);
	await page.getByRole('button', { name: 'Workspaces', exact: true }).click();
	await page.getByRole('button', { name: 'Create workspace', exact: true }).click();
	const workspaceSlug = page.getByRole('dialog').getByLabel('Workspace URL', { exact: true });
	await workspaceSlug.pressSequentially('my-team-');
	await expect(workspaceSlug).toHaveValue('my-team-');
	await workspaceSlug.press('Tab');
	await expect(workspaceSlug).toHaveValue('my-team');
});

test('workspace setup retries a failed list without redirecting an authenticated user', async ({ page }) => {
	let attempts = 0;
	await page.route('**://*/api/**', (route) => {
		const path = new URL(route.request().url()).pathname;
		if (path === '/api/auth/me') return route.fulfill({ json: user });
		if (path === '/api/workspaces') {
			attempts++;
			return attempts === 1
				? route.fulfill({ status: 503, json: { error: { message: 'Unavailable' } } })
				: route.fulfill({ json: [] });
		}
		return route.fulfill({ json: [] });
	});
	await page.goto('/workspace-setup');
	await expect(page.getByRole('alert')).toContainText('Could not load your workspace');
	await page.getByRole('button', { name: 'Retry' }).click();
	await expect(page.getByRole('heading', { name: 'Set up your workspace' })).toBeVisible();
	await expect(page).toHaveURL(/\/workspace-setup$/);
	const workspaceSlug = page.getByLabel('Workspace URL slug');
	await workspaceSlug.fill('');
	await workspaceSlug.pressSequentially('my-team-');
	await expect(workspaceSlug).toHaveValue('my-team-');
	await workspaceSlug.press('Tab');
	await expect(workspaceSlug).toHaveValue('my-team');
});
