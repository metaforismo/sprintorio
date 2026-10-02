import { defineConfig } from '@playwright/test';

export default defineConfig({
	testDir: 'tests/e2e',
	timeout: process.env.PLAYWRIGHT_DEV_SERVER === '1' ? 120_000 : 30_000,
	forbidOnly: !!process.env.CI,
	retries: 0,
	workers: process.env.CI ? 2 : undefined,
	reporter: process.env.CI ? [['dot'], ['html', { open: 'never' }]] : 'list',
	webServer: {
		command:
			process.env.PLAYWRIGHT_DEV_SERVER === '1'
				? 'npm run dev -- --port 4174'
				: process.env.PLAYWRIGHT_PREBUILT === '1'
				? 'npm run preview -- --port 4174'
				: 'npm run build && npm run preview -- --port 4174',
		port: 4174,
		reuseExistingServer: false,
		timeout: 480_000
	},
	use: {
		baseURL: 'http://localhost:4174',
		channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure'
	}
});
