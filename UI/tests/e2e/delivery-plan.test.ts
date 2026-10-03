import { expect, test, type Page, type Dialog } from '@playwright/test';

const emptyPlan = () => ({
	product_name: '',
	objective: '',
	success_metric: '',
	target_release: '',
	milestones: [],
	test_cases: []
});
async function setup(page: Page, role = 'member', includeSecondProject = false) {
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
		if (path.endsWith('/teams') && includeSecondProject)
			return route.fulfill({ json: [{ id: 't1', name: 'Product', key: 'PRD' }] });
		if (path === '/api/workspaces/test/projects/p2')
			return route.fulfill({ json: { id: 'p2', name: 'Another project', status: 'planned', team_id: null } });
		if (path === '/api/workspaces/test/projects')
			return route.fulfill({
				json: [
					{ id: 'p1', name: 'Launch project', status: 'planned' },
					...(includeSecondProject ? [{ id: 'p2', name: 'Another project', status: 'planned', team_id: 't1' }] : [])
				]
			});
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

async function choose(page: Page, label: string, option: string) {
	await page.getByRole('button', { name: label, exact: true }).click();
	await page.getByRole('option', { name: option, exact: true }).click();
}
async function apply(page: Page) {
	await page.getByRole('dialog').getByRole('button', { name: 'Apply changes', exact: true }).click();
	await expect(page.getByRole('dialog')).toHaveCount(0);
}
async function template(page: Page, name: string) {
	await page.getByRole('button', { name: 'Test templates', exact: true }).click();
	await page.getByRole('menuitem', { name, exact: true }).click();
	await apply(page);
}
const savedTest = (title = 'Order status visible') => ({
	id: 'test-1',
	title,
	steps: 'Sign in and open an order',
	expected_result: 'Current status appears',
	status: 'passed',
	evidence: 'Checked in Firefox; status was Delivered.'
});

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
	await choose(page, 'Milestone status', 'Done');
	await page.getByRole('button', { name: 'Add test case' }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByLabel('Test case', { exact: true }).fill('Order status visible');
	await dialog.getByLabel('Steps', { exact: true }).fill('Sign in and open an order');
	await dialog.getByLabel('Expected result', { exact: true }).fill('Current status appears');
	await choose(page, 'Test status', 'Passed');
	await dialog.getByRole('button', { name: 'Apply changes' }).click();
	await expect(dialog).toBeVisible();
	expect(state.writes).toHaveLength(0);
	await expect(dialog.getByLabel('Evidence *', { exact: true })).toHaveAttribute('required', '');
	await dialog.getByLabel('Evidence *', { exact: true }).fill('Checked in Firefox; status was Delivered.');
	await apply(page);
	await expect(page.getByTestId('delivery-readiness')).toContainText('Ready for review');
	// Same-project tab changes preserve the local plan.
	const unexpectedPrompts: string[] = [];
	const rejectUnexpectedPrompt = async (prompt: Dialog) => {
		unexpectedPrompts.push(prompt.message());
		await prompt.dismiss();
	};
	page.on('dialog', rejectUnexpectedPrompt);
	await page.getByRole('button', { name: 'Issue list' }).click();
	await page.getByRole('button', { name: 'Delivery', exact: true }).click();
	await expect(page.getByLabel('Product name', { exact: true })).toHaveValue('Customer portal');
	expect(unexpectedPrompts).toEqual([]);
	page.off('dialog', rejectUnexpectedPrompt);
	await page.getByRole('button', { name: 'Save plan' }).click();
	await expect(page.locator('.app-toast').filter({ hasText: 'Delivery plan saved.' })).toBeVisible();
	expect(state.writes[0]).toMatchObject({
		version: 0,
		plan: {
			product_name: 'Customer portal',
			target_release: '2026-11-01',
			test_cases: [{ status: 'passed', evidence: 'Checked in Firefox; status was Delivered.' }]
		}
	});
	await page.reload();
	await page.getByRole('button', { name: 'Open test: Order status visible', exact: true }).click();
	await dialog.getByLabel('Expected result', { exact: true }).fill('An updated acceptance criterion');
	await expect(dialog.getByRole('button', { name: 'Test status' })).toHaveText('Not run');
	await expect(dialog.getByLabel('Evidence', { exact: true })).toHaveValue('');
	await apply(page);
	await expect(page.getByTestId('delivery-readiness')).toContainText('Testing pending');
	await page.getByRole('button', { name: 'Cancel changes' }).click();
	await page.getByRole('button', { name: 'Open test: Order status visible', exact: true }).click();
	await choose(page, 'Test status', 'Failed');
	await apply(page);
	await expect(page.getByTestId('delivery-readiness')).toContainText('Changes needed');
	await page.getByRole('button', { name: 'Cancel changes' }).click();
	await expect(page.getByTestId('test-row')).toContainText('Passed');
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

test('guest can read test details but cannot edit or save a delivery plan', async ({ page }) => {
	const state = await setup(page, 'guest');
	state.latest({ ...emptyPlan(), test_cases: [savedTest()] });
	await page.reload();
	await expect(page.getByText('View only.', { exact: false })).toBeVisible();
	await expect(page.getByLabel('Product name', { exact: true })).toBeDisabled();
	await expect(page.getByRole('button', { name: 'Save plan' })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Add milestone' })).toHaveCount(0);
	await page.getByRole('button', { name: 'Open test: Order status visible' }).click();
	await expect(page.getByRole('dialog').getByLabel('Test case', { exact: true })).toBeDisabled();
	await expect(page.getByRole('button', { name: 'Apply changes' })).toHaveCount(0);
	await page.getByRole('dialog').locator('form').getByRole('button', { name: 'Close', exact: true }).click();
	expect(state.writes).toHaveLength(0);
});

test('delivery form and test editor fit a narrow mobile viewport', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await setup(page);
	await page.getByLabel('Product name', { exact: true }).fill('Mobile portal');
	await page.getByRole('button', { name: 'Add milestone' }).click();
	await page.getByLabel('Milestone 1', { exact: true }).fill('Customer rollout');
	await page.getByRole('button', { name: 'Add test case' }).click();
	await page.getByLabel('Test case', { exact: true }).fill('Mobile sign in');
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
	const box = await page.getByRole('dialog').boundingBox();
	expect(box!.x).toBeGreaterThanOrEqual(0);
	expect(box!.x + box!.width).toBeLessThanOrEqual(390);
	await page.getByRole('button', { name: 'Cancel', exact: true }).click();
});

test('manual test templates stay unrun and saved test lists can be searched and filtered', async ({ page }) => {
	const state = await setup(page);
	for (const name of ['Acceptance', 'Regression', 'Accessibility', 'Acceptance']) await template(page, name);
	await expect(page.getByTestId('delivery-readiness')).toContainText('Testing pending');
	await page.getByRole('button', { name: 'Save plan', exact: true }).click();
	await expect.poll(() => state.writes.length).toBe(1);
	expect(state.writes[0].plan.test_cases.every((item: any) => item.status === 'not_run' && !item.evidence)).toBe(true);
	await page.reload();
	await expect(page.getByLabel('Steps', { exact: true })).toHaveCount(0);
	await page.getByLabel('Search tests', { exact: true }).fill('keyboard');
	await expect(page.getByRole('button', { name: 'Open test: Accessibility check', exact: true })).toHaveCount(1);
	await expect(page.getByRole('button', { name: 'Open test: Acceptance check', exact: true })).toHaveCount(0);
	await choose(page, 'Filter by result', 'Passed');
	await expect(page.getByText('No matching tests.', { exact: true })).toBeVisible();
	await choose(page, 'Filter by result', 'All results');
	await page.getByRole('button', { name: 'Test actions: Accessibility check', exact: true }).click();
	await page.getByRole('menuitem', { name: 'Remove test', exact: true }).click();
	await expect(page.getByLabel('Search tests', { exact: true })).toHaveValue('keyboard');
	await page.getByLabel('Search tests', { exact: true }).fill('');
	await expect(page.getByRole('button', { name: 'Open test: Acceptance check', exact: true })).toHaveCount(2);
	await expect(page.getByRole('button', { name: 'Open test: Regression check', exact: true })).toHaveCount(1);
});

test('test dialog cancel preserves the plan and duplication resets recorded outcome', async ({ page }) => {
	const state = await setup(page);
	state.latest({ ...emptyPlan(), test_cases: [savedTest()] });
	await page.reload();
	await page.getByRole('button', { name: 'Open test: Order status visible', exact: true }).click();
	await page.getByLabel('Test case', { exact: true }).fill('Discard this edit');
	await page.getByRole('dialog').getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Save plan' })).toBeDisabled();
	await expect(page.getByTestId('test-row')).toContainText('Order status visible');
	await page.getByRole('button', { name: 'Test actions: Order status visible', exact: true }).click();
	await page.getByRole('menuitem', { name: 'Duplicate test', exact: true }).click();
	await expect(page.getByLabel('Test case', { exact: true })).toHaveValue('Order status visible (Copy)');
	await expect(page.getByRole('button', { name: 'Test status' })).toHaveText('Not run');
	await expect(page.getByLabel('Evidence', { exact: true })).toHaveValue('');
	await apply(page);
	await page.getByRole('button', { name: 'Save plan' }).click();
	await expect.poll(() => state.writes.length).toBe(1);
	expect(state.writes[0].plan.test_cases).toHaveLength(2);
	expect(state.writes[0].plan.test_cases[0]).toMatchObject({ status: 'passed', evidence: savedTest().evidence });
	expect(state.writes[0].plan.test_cases[1]).toMatchObject({ status: 'not_run', evidence: '' });
});

test('copied agent context stays saved and repeated toast feedback preserves popover geometry', async ({ page }) => {
	await page.addInitScript(() => {
		let attempts = 0;
		Object.defineProperty(navigator, 'clipboard', {
			configurable: true,
			value: {
				writeText: async (text: string) => {
					if (++attempts % 2 === 1) throw new Error('Clipboard unavailable');
					(window as any).__deliveryClipboard = text;
				}
			}
		});
	});
	const state = await setup(page);
	state.latest({
		...emptyPlan(),
		product_name: 'Saved brief',
		test_cases: Array.from({ length: 12 }, (_, i) => ({
			id: `t${i}`,
			title: `Check ${i}`,
			steps: 'Perform check',
			expected_result: 'Correct result',
			status: 'not_run',
			evidence: ''
		}))
	});
	await page.reload();
	const copy = page.getByRole('button', { name: 'Copy context', exact: true });
	const geometry = await copy.boundingBox();
	for (let i = 0; i < 20; i++) {
		await copy.click();
		const message = i % 2 === 0 ? 'Could not copy. Try again.' : 'Saved context copied.';
		await expect(page.locator('.app-toast').filter({ hasText: message })).toBeVisible();
		await expect(page.locator('.app-toast')).toHaveCount(1);
		expect(await copy.boundingBox()).toEqual(geometry);
		await page.getByRole('button', { name: 'How readiness works', exact: true }).click();
		const popover = page.locator('[data-slot="popover-content"]');
		await expect(popover).toBeVisible();
		const box = await popover.boundingBox();
		const viewport = page.viewportSize()!;
		expect(box!.x).toBeGreaterThanOrEqual(0);
		expect(box!.x + box!.width).toBeLessThanOrEqual(viewport.width);
		await page.keyboard.press('Escape');
		await expect(popover).toHaveCount(0);
	}
	const context = JSON.parse(await page.evaluate(() => (window as any).__deliveryClipboard));
	expect(context).toMatchObject({
		workspace: 'test',
		project_id: 'p1',
		version: 1,
		basis: 'saved_manual_verification',
		brief: { product: 'Saved brief' },
		tests: { total: 12, not_run: 12, attention_omitted: 2 }
	});
	await page.getByLabel('Product name', { exact: true }).fill('Unsaved brief');
	await expect(copy).toBeDisabled();
	await page.getByRole('button', { name: 'Cancel changes', exact: true }).click();
	await expect(copy).toBeEnabled();
});

test('dirty delivery view history stays in the project while leaving requires confirmation', async ({ page }) => {
	await setup(page, 'member', true);
	await page.getByLabel('Product name', { exact: true }).fill('Keep this local draft');
	const prompts: string[] = [];
	let allowLeaving = false;
	page.on('dialog', async (prompt) => {
		prompts.push(prompt.message());
		if (allowLeaving) await prompt.accept();
		else await prompt.dismiss();
	});
	await page.getByRole('button', { name: 'Issue list' }).click();
	await expect(page).toHaveURL(/projects\/p1$/);
	await page.goBack();
	await expect(page).toHaveURL(/projects\/p1\?view=delivery$/);
	await expect(page.getByLabel('Product name', { exact: true })).toHaveValue('Keep this local draft');
	await page.goForward();
	await expect(page).toHaveURL(/projects\/p1$/);
	await page.getByRole('button', { name: 'Project actions' }).click();
	await page.getByRole('menuitem', { name: 'Testing', exact: true }).click();
	await expect(page).toHaveURL(/projects\/p1\?view=delivery&section=testing$/);
	expect(prompts).toEqual([]);
	const sidebar = page.getByRole('complementary');
	const anotherProject = sidebar.getByRole('link', { name: 'Another project', exact: true });
	await anotherProject.click();
	await expect.poll(() => prompts.length).toBe(1);
	expect(prompts[0]).toBe('Leave this project and discard unsaved delivery changes?');
	await expect(page).toHaveURL(/projects\/p1\?view=delivery&section=testing$/);
	await expect(page.getByLabel('Product name', { exact: true })).toHaveValue('Keep this local draft');
	allowLeaving = true;
	await anotherProject.click();
	await expect(page).toHaveURL(/projects\/p2$/);
	expect(prompts).toHaveLength(2);
});
