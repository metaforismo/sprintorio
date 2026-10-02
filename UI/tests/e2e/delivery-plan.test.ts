import { expect, test, type Page } from '@playwright/test';
import { mkdir, copyFile } from 'node:fs/promises';

const emptyPlan = () => ({
	product_name: '',
	objective: '',
	success_metric: '',
	target_release: '',
	milestones: [],
	test_cases: []
});
async function setup(page: Page, role = 'member') {
	let plan: any = emptyPlan();
	let version = 0;
	let conflict = false;
	const writes: any[] = [];
	await page.route('**://*/api/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		if (path.endsWith('/delivery-plan')) {
			if (route.request().method() === 'PATCH') {
				if (conflict) return route.fulfill({ status: 409, json: { error: { code: 'DELIVERY_PLAN_CONFLICT' } } });
				const body = route.request().postDataJSON();
				writes.push(body);
				plan = body.plan;
				version++;
			}
			return route.fulfill({ json: { plan, version } });
		}
		if (path === '/api/auth/me')
			return route.fulfill({ json: { id: 'u', email: 'member@example.com', name: 'Member' } });
		if (path === '/api/preferences')
			return route.fulfill({
				json: { font_size: 'default', theme_mode: 'dark', dark_theme: 'dark', workflow_sort_order: [] }
			});
		if (path === '/api/workspaces') return route.fulfill({ json: [{ id: 'w', name: 'Workspace', slug: 'test' }] });
		if (path === '/api/workspaces/test')
			return route.fulfill({ json: { id: 'w', name: 'Workspace', slug: 'test', current_user_role: role } });
		if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
		if (path === '/api/workspaces/test/projects/p1')
			return route.fulfill({ json: { id: 'p1', name: 'Launch project', status: 'planned', team_id: null } });
		if (path === '/api/workspaces/test/projects')
			return route.fulfill({ json: [{ id: 'p1', name: 'Launch project', status: 'planned' }] });
		if (path.endsWith('/issues'))
			return route.fulfill({ json: { data: [], total_count: 0, has_more: false, page: 1 } });
		if (path.endsWith('/github/status')) return route.fulfill({ json: { configured: false, repos: [] } });
		if (path.endsWith('/dev-machine-scope-setting')) return route.fulfill({ json: {} });
		return route.fulfill({ json: [] });
	});
	await page.goto('/test/projects/p1');
	await page.getByRole('button', { name: 'Delivery', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Product delivery' })).toBeVisible();
	return {
		writes,
		setConflict: () => {
			conflict = true;
		},
		latest: (value: any) => {
			plan = value;
			version++;
		}
	};
}

test('member persists product, milestones and manual test evidence; readiness stays conservative', async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 1600 });
	const state = await setup(page);
	await expect(page.getByText('No tests recorded', { exact: true })).toBeVisible();
	await page.getByLabel('Product name', { exact: true }).fill('Customer portal');
	await page.getByLabel('Objective', { exact: true }).fill('Let customers track orders');
	await page.getByLabel('Success metric', { exact: true }).fill('Customers can find their order status');
	await page.getByLabel('Target release', { exact: true }).fill('2026-11-01');
	await page.getByRole('button', { name: 'Add milestone' }).click();
	await page.getByLabel('Milestone 1', { exact: true }).fill('Portal shipped');
	await page.getByLabel('Milestone status').selectOption('done');
	await page.getByRole('button', { name: 'Add test case' }).click();
	await page.getByLabel('Test case 1', { exact: true }).fill('Order status visible');
	await page.getByLabel('Steps', { exact: true }).fill('Sign in and open an order');
	await page.getByLabel('Expected result', { exact: true }).fill('Current status appears');
	await expect(page.getByText('Testing pending', { exact: true })).toBeVisible();
	await page.getByLabel('Test status').selectOption('passed');
	await page.getByRole('button', { name: 'Save plan' }).click();
	expect(state.writes).toHaveLength(0); // browser blocks a passed result without evidence
	await page.getByLabel('Evidence (required)', { exact: true }).fill('Checked in Firefox; status was Delivered.');
	await expect(page.getByText('Ready for review', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Issue list' }).click();
	await page.getByRole('button', { name: 'Delivery', exact: true }).click();
	await expect(page.getByLabel('Product name', { exact: true })).toHaveValue('Customer portal');
	await page.getByRole('button', { name: 'Save plan' }).click();
	await expect(page.getByRole('status').filter({ hasText: 'Delivery plan saved.' })).toBeVisible();
	if (process.env.UPDATE_SCREENSHOTS === '1') {
		await mkdir('../assets', { recursive: true });
		await mkdir('../WEB/static', { recursive: true });
		await page.screenshot({ path: '../assets/product-screenshot.png', fullPage: true });
		await copyFile('../assets/product-screenshot.png', '../WEB/static/product-screenshot.png');
		await page.screenshot({ path: '../WEB/static/product-screenshot-1440.png', fullPage: true });
		await page.setViewportSize({ width: 720, height: 1600 });
		await page.screenshot({ path: '../WEB/static/product-screenshot-720.png', fullPage: true });
		await page.setViewportSize({ width: 1440, height: 1600 });
	}
	expect(state.writes[0]).toMatchObject({
		version: 0,
		plan: {
			product_name: 'Customer portal',
			target_release: '2026-11-01',
			test_cases: [{ status: 'passed', evidence: 'Checked in Firefox; status was Delivered.' }]
		}
	});
	await page.reload();
	await page.getByRole('button', { name: 'Delivery', exact: true }).click();
	await expect(page.getByLabel('Product name', { exact: true })).toHaveValue('Customer portal');
	await page.getByLabel('Expected result', { exact: true }).fill('An updated acceptance criterion');
	await expect(page.getByLabel('Test status')).toHaveValue('not_run');
	await expect(page.getByLabel('Evidence', { exact: true })).toHaveValue('');
	await expect(page.getByText('Testing pending', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Cancel changes' }).click();
	await page.getByLabel('Test status').selectOption('failed');
	await expect(page.getByText('Changes needed', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Cancel changes' }).click();
	await expect(page.getByLabel('Test status')).toHaveValue('passed');
});

test('conflict keeps unsaved edits until an explicit reload', async ({ page }) => {
	const state = await setup(page);
	await page.getByLabel('Product name', { exact: true }).fill('My local draft');
	state.latest({ ...emptyPlan(), product_name: 'Newer teammate version' });
	state.setConflict();
	await page.getByRole('button', { name: 'Save plan' }).click();
	await expect(page.getByRole('alert').filter({ hasText: 'Your edits are still here' })).toBeVisible();
	await expect(page.getByLabel('Product name', { exact: true })).toHaveValue('My local draft');
	await expect(page.getByRole('button', { name: 'Save plan' })).toBeDisabled();
	await page.getByRole('button', { name: 'Reload latest and discard my edits' }).click();
	await expect(page.getByLabel('Product name', { exact: true })).toHaveValue('Newer teammate version');
});

test('guest can read but cannot edit or save a delivery plan', async ({ page }) => {
	const state = await setup(page, 'guest');
	await expect(page.getByText('View only.', { exact: false })).toBeVisible();
	await expect(page.getByLabel('Product name', { exact: true })).toBeDisabled();
	await expect(page.getByRole('button', { name: 'Save plan' })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Add milestone' })).toHaveCount(0);
	expect(state.writes).toHaveLength(0);
});

test('delivery form works on a narrow mobile viewport without horizontal overflow', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await setup(page);
	await page.getByLabel('Product name', { exact: true }).fill('Mobile portal');
	await page.getByRole('button', { name: 'Add milestone' }).click();
	await page.getByLabel('Milestone 1', { exact: true }).fill('Customer rollout');
	await page.getByRole('button', { name: 'Add test case' }).click();
	await page.getByLabel('Test case 1', { exact: true }).fill('Mobile sign in');
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
	if (process.env.UPDATE_SCREENSHOTS === '1') {
		await mkdir('../assets', { recursive: true });
		await page.screenshot({ path: '../assets/delivery-mobile.png', fullPage: true });
	}
});

test('manual test templates stay unrun and saved test lists can be searched and filtered', async ({ page }) => {
	const state = await setup(page);
	for (const name of ['Acceptance', 'Regression', 'Accessibility', 'Acceptance']) {
		await page.getByRole('button', { name, exact: true }).click();
	}
	await expect(page.getByText('Testing pending', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Save plan', exact: true }).click();
	expect(state.writes[0].plan.test_cases.every((item: any) => item.status === 'not_run' && !item.evidence)).toBe(true);
	await page.reload();
	await page.getByRole('button', { name: 'Delivery', exact: true }).click();
	await expect(page.getByLabel('Steps', { exact: true }).first()).toBeHidden();
	await page.getByLabel('Search tests', { exact: true }).fill('keyboard');
	await expect(page.locator('summary').filter({ hasText: 'Accessibility check' })).toHaveCount(1);
	await expect(page.locator('summary').filter({ hasText: 'Acceptance check' })).toHaveCount(0);
	const accessibilitySummary = page.locator('summary').filter({ hasText: 'Accessibility check' });
	if ((await accessibilitySummary.locator('..').getAttribute('open')) === null) await accessibilitySummary.click();
	await expect(page.getByLabel('Test case 3', { exact: true })).toHaveValue('Accessibility check');

	await page.getByLabel('Filter by result', { exact: true }).selectOption('passed');
	await expect(page.getByText('No matching tests.', { exact: true })).toBeVisible();
	await page.getByLabel('Filter by result', { exact: true }).selectOption('all');
	if ((await accessibilitySummary.locator('..').getAttribute('open')) === null) await accessibilitySummary.click();
	await page.getByRole('button', { name: 'Remove test case 3', exact: true }).click();
	await expect(page.getByLabel('Search tests', { exact: true })).toHaveValue('keyboard');
	await page.getByLabel('Search tests', { exact: true }).fill('');
	await expect(page.locator('summary').filter({ hasText: 'Acceptance check' })).toHaveCount(2);
	await expect(page.locator('summary').filter({ hasText: 'Regression check' })).toHaveCount(1);
});
