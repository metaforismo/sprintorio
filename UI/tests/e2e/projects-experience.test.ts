import { expect, test, type Page } from '@playwright/test';

const project = {
	id: 'p1',
	name: 'Portal launch',
	status: 'planned',
	team_id: 't1',
	progress: { total: 4, completed: 1, cancelled: 2 }
};
async function setup(page: Page, options: { failList?: boolean; failCreate?: boolean } = {}) {
	let listFailed = !!options.failList;
	let createFailed = false;
	const writes: string[] = [];
	const cycles: string[] = [];
	await page.route('**://*/api/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		const method = route.request().method();
		if (path === '/api/auth/me')
			return route.fulfill({ json: { id: 'u1', name: 'Member', email: 'member@example.com' } });
		if (path === '/api/preferences')
			return route.fulfill({ json: { theme_mode: 'dark', font_size: 'default', workflow_sort_order: [] } });
		if (path === '/api/workspaces') return route.fulfill({ json: [{ id: 'w', slug: 'test', name: 'Workspace' }] });
		if (path === '/api/workspaces/test')
			return route.fulfill({ json: { id: 'w', slug: 'test', name: 'Workspace', current_user_role: 'member' } });
		if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
		if (path.endsWith('/teams'))
			return route.fulfill({
				json: [
					{ id: 't1', name: 'Product', key: 'PRD' },
					{ id: 't2', name: 'Other', key: 'OTH' }
				]
			});
		if (path.endsWith('/projects') && method === 'POST') {
			writes.push(route.request().postDataJSON().name);
			if (options.failCreate && !createFailed) {
				createFailed = true;
				return route.fulfill({ status: 500, json: { error: { code: 'INTERNAL_ERROR', message: 'Retry creation' } } });
			}
			return route.fulfill({ json: project });
		}
		if (path.endsWith('/projects')) {
			if (listFailed) {
				return route.fulfill({ status: 500, json: { error: { code: 'INTERNAL_ERROR' } } });
			}
			return route.fulfill({ json: [project] });
		}
		if (path.endsWith('/projects/p1')) {
			if (method === 'DELETE') writes.push('DELETE');
			return route.fulfill({ json: project });
		}
		if (path.endsWith('/delivery-plan'))
			return route.fulfill({
				json: {
					plan: {
						product_name: '',
						objective: '',
						success_metric: '',
						target_release: '',
						milestones: [],
						test_cases: []
					},
					version: 0
				}
			});
		if (path.endsWith('/issues'))
			return route.fulfill({ json: { data: [], total_count: 0, has_more: false, page: 1 } });
		if (path.endsWith('/cycles')) cycles.push(path);
		if (path.endsWith('/github/status')) return route.fulfill({ json: { configured: false, repos: [] } });
		if (path.endsWith('/dev-machine-scope-setting')) return route.fulfill({ json: {} });
		return route.fulfill({ json: [] });
	});
	return { writes, cycles, allowList: () => { listFailed = false; } };
}

test('project creation retains a failed draft, retries once, and progress excludes cancelled work', async ({
	page
}) => {
	const state = await setup(page, { failCreate: true });
	await page.goto('/test/projects');
	await expect(page.getByText('25%', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'New Project', exact: true }).click();
	await page.getByLabel('Name', { exact: true }).fill('Customer portal');
	await page.getByRole('dialog').getByRole('button', { name: 'Create project', exact: true }).click();
	await expect(page.getByRole('dialog')).toBeVisible();
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Customer portal');
	await expect(page.getByRole('dialog').getByRole('alert')).toBeVisible();
	await page.getByRole('dialog').getByRole('button', { name: 'Create project', exact: true }).click();
	await expect(page.getByRole('dialog')).toHaveCount(0);
	expect(state.writes).toEqual(['Customer portal', 'Customer portal']);
});

test('project list error has retry; timeline loads only the project team and deletion requires confirmation', async ({
	page
}) => {
	const state = await setup(page, { failList: true });
	await page.goto('/test/projects');
	await expect(page.getByRole('alert').filter({ hasText: 'Could not load projects' })).toBeVisible();
	state.allowList();
	await page.getByRole('button', { name: 'Retry', exact: true }).click();
	await page.getByRole('link', { name: /Portal launch/ }).click();
	await expect(page.getByRole('button', { name: 'Issue list' })).toBeVisible();
	expect(state.cycles).toEqual([]);
	await page.getByRole('button', { name: 'Gantt chart' }).click();
	await expect.poll(() => state.cycles).toEqual(['/api/workspaces/test/teams/t1/cycles']);
	await page.getByRole('button', { name: 'Project actions' }).click();
	await page.getByRole('button', { name: 'Delete project', exact: true }).click();
	await expect(page.getByRole('dialog')).toContainText('Portal launch');
	expect(state.writes).not.toContain('DELETE');
	await page.getByRole('dialog').getByRole('button', { name: 'Cancel', exact: true }).click();
	expect(state.writes).not.toContain('DELETE');
});
