import { expect, test, type Page } from '@playwright/test';

async function workspace(page: Page, searchHandler?: Parameters<Page['route']>[1]) {
	if (searchHandler) await page.route('**/api/workspaces/test/issues?*', searchHandler);
	await page.route('**/api/**', async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname.endsWith('/issues') && url.searchParams.has('search')) return route.fallback();
		const data = url.pathname === '/api/auth/me' ? { id: 'u', name: 'QA', email: 'qa@example.test', dev_machines_enabled: false }
			: url.pathname === '/api/preferences' ? { theme_mode: 'dark', font_size: 'default', workflow_sort_order: [] }
			: url.pathname === '/api/workspaces' ? [{ id: 'w', slug: 'test', name: 'QA' }]
			: url.pathname === '/api/workspaces/test' ? { id: 'w', slug: 'test', name: 'QA', current_user_role: 'owner' }
			: url.pathname.endsWith('/teams') ? [{ id: 't', key: 'ENG', name: 'Développement' }]
			: url.pathname.endsWith('/issues') ? { data: [], total_count: 0, has_more: false, page: 1 }
			: url.pathname === '/api/notifications' ? { notifications: [], unread_count: 0 } : [];
		return route.fulfill({ json: data });
	});
	await page.goto('/test/my-issues');
	await page.getByRole('button', { name: 'Search', exact: true }).click();
	return page.getByRole('combobox', { name: /Type a command/ });
}
const issue = (id: string, title: string) => ({ id, identifier: id, title, status: 'todo', priority: 0, team_id: 't' });

test('late search results cannot overwrite a newer query and clearing cancels queued searches', async ({ page }) => {
	let releaseOld!: () => void;
	let oldStarted!: () => void;
	const started = new Promise<void>((resolve) => { oldStarted = resolve; });
	const old = new Promise<void>((resolve) => { releaseOld = resolve; });
	const input = await workspace(page, async (route) => {
		const query = new URL(route.request().url()).searchParams.get('search');
		if (query === 'older') { oldStarted(); await old; }
		await route.fulfill({ json: { data: [issue(query === 'older' ? 'ENG-1' : 'ENG-2', query === 'older' ? 'Older response' : 'Newest response')], has_more: false, total_count: 1, page: 1 } });
	});
	try {
		await input.fill('older');
		await started;
		await input.fill('newest');
		await expect(page.getByRole('option').filter({ hasText: 'Newest response' })).toBeVisible();
	} finally { releaseOld(); }
	await expect(page.getByRole('option').filter({ hasText: 'Older response' })).toHaveCount(0);
	await input.press('Escape');
	await expect(input).toHaveValue('');
	await expect(input).toBeFocused();
	await input.press('Escape');
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Search', exact: true })).toBeFocused();
});

test('search distinguishes server errors from empty results and commands match accents or team keys', async ({ page }) => {
	let attempts = 0;
	const input = await workspace(page, async (route) => {
		attempts++;
		if (attempts === 1) return route.fulfill({ status: 503, json: { error: { code: 'UNAVAILABLE' } } });
		return route.fulfill({ json: { data: [], has_more: false, total_count: 0, page: 1 } });
	});
	await input.fill('missing');
	await expect(page.getByRole('alert').filter({ hasText: 'Search unavailable.' })).toBeVisible();
	await page.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(page.getByRole('alert').filter({ hasText: 'Search unavailable.' })).toHaveCount(0);
	await input.fill('developpement');
	await expect(page.getByRole('option').filter({ hasText: 'Développement' })).toBeVisible();
	await input.fill('ENG');
	await expect(page.getByRole('option').filter({ hasText: 'Développement' })).toBeVisible();
	await page.keyboard.press('Tab');
	await expect(page.locator('[role=dialog]')).toContainText('Enter');
});

test('confirming IME composition does not execute the selected command', async ({ page }) => {
 const input = await workspace(page, route => route.fulfill({ json: { data: [], has_more: false, total_count: 0, page: 1 } }));
 await input.fill('projects');
 await expect(page.getByRole('option').filter({ hasText: 'Projects' })).toBeVisible();
 await input.dispatchEvent('keydown', { key: 'Enter', code: 'Enter', isComposing: true });
 await expect(page.getByRole('dialog')).toBeVisible();
 await expect(input).toBeFocused();
 await expect(page).toHaveURL(/\/test\/my-issues$/);
 await input.press('Enter');
 await expect(page).toHaveURL(/\/test\/projects$/);
});
