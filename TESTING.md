# Testing Sprintorio

## Fast checks

```sh
make check-brand       # Identity, links, module path, and asset checks
make test-unit         # Go packages and frontend unit tests
make check             # Svelte/TypeScript checks for app and site
```

Install UI and WEB dependencies with `npm ci` first. Go uses the version declared in `BE/go.mod`. Some backend integration tests require `DATABASE_URL`; without it they explicitly skip. A green unit run alone does not establish database integration coverage.

## PostgreSQL integration

Use a dedicated, disposable PostgreSQL database. Never point tests at production.

```sh
export DATABASE_URL='postgres://test_user:test_password@localhost:5432/sprintorio_test?sslmode=disable'
cd BE
go run ./cmd/server migrate up
go test ./... -race -count=1 -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

The suite checks persisted plans, workspace isolation, concurrent versioned saves, project metadata/date validation, nullable updates, team/status rollback, nested team-resource scope, migration behavior, and workspace export/import. Import keeps local milestone/test IDs intact and remaps application asset URLs. Version conflicts return HTTP 409 instead of silently overwriting another save.

## Browser checks

```sh
cd UI
npx playwright install chromium
npm run test:e2e
cd ../WEB
npm run validate
npm run test:responsive
```

The UI browser suite mocks API responses to exercise forms, failed-draft retry, permissions, conflict recovery, selector keyboard/focus behavior, bounded session recovery, project navigation, filtered test deletion, and readiness states. Backend integration tests separately exercise actual PostgreSQL persistence. Mocked browser fixtures and screenshots are examples; they are not live customer data or proof of a full application deployment.

For an already installed Chrome browser, use `PLAYWRIGHT_CHANNEL=chrome` with the browser-test commands.

To run browser checks against Vite development mode instead of a production preview, set `PLAYWRIGHT_DEV_SERVER=1`. Production build validation remains a separate requirement.

After a successful `npm run build` on the same checkout, set `PLAYWRIGHT_PREBUILT=1` to reuse that build for browser checks instead of rebuilding it.

Documentation screenshots use sample data and are updated only when explicitly requested:

```sh
cd UI
UPDATE_SCREENSHOTS=1 PLAYWRIGHT_DEV_SERVER=1 PLAYWRIGHT_CHANNEL=chrome npm run test:e2e -- tests/e2e/delivery-plan.test.ts --workers=1
```

## Manual release review

In **Project → Delivery**, define the product objective and success metric, add milestones, and record manual test steps and expected results. Record evidence for each passed or failed check. Empty, pending, failed, and blocked tests prevent the readiness summary from implying the plan is complete. Editing test definitions invalidates previous results; repeat the check before recording a new result.

These records do not execute tests or authorize deployment. Automated tests run through development commands and GitHub Actions.

## CI

GitHub Actions run identity checks, Go race/coverage tests with PostgreSQL, frontend checks and Playwright tests, and marketing-site build/SEO/responsive checks. A separate container integration job exercises Dev Machines on a Linux host with XFS project quotas. That runtime cannot be verified by ordinary frontend or Go unit tests.
