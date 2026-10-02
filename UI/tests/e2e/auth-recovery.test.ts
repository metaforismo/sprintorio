import { test, expect } from '@playwright/test';

test('a rejected session after refresh returns to sign-in without a request loop', async ({ page }) => {
	let sessionRequests = 0;
	let refreshRequests = 0;
	await page.route('**://*/api/auth/me', async (route) => {
		sessionRequests++;
		await route.fulfill({ status: 401, json: { error: { code: 'UNAUTHORIZED', message: 'Unauthorized' } } });
	});
	await page.route('**://*/api/auth/refresh', async (route) => {
		refreshRequests++;
		await route.fulfill({ json: { status: 'refreshed' } });
	});
	await page.goto('/');
	await expect(page).toHaveURL(/\/login$/);
	await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeVisible();
	expect(sessionRequests).toBe(2);
	expect(refreshRequests).toBe(1);
});
