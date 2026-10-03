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

## Agent clients and MCP

```sh
cd BE
go test ./internal/agentclient ./cmd/sprintorio -race -count=1
go test ./internal/middleware ./internal/service ./internal/repository -race -count=1
```

The client suite uses local HTTP fixtures and a child process with real stdin/stdout. It checks compact/explicit-field output, pagination, strict operation inputs, validation before writes, delivery conflicts, credential redaction, redirect refusal, message/body limits, file upload/export, JSON-RPC lifecycle, stdio hygiene, remote caller authorization, exact Origin allowlists and formatted HTTP JSON. Its measured three-tool catalogue is 879 JSON bytes; byte size is not a tokenizer-specific token count.

Backend tests separately cover agent-token scope, expiry/revocation, current workspace roles and cross-workspace references. Database-backed tests still require the disposable `DATABASE_URL` above. A fixture result does not establish a real-account connection.

For a real API smoke, use a disposable workspace: create a token in Settings → Agents, read context/statuses, create a project and issue, read the full delivery plan with `detail:true`, save using its version, and verify a stale save conflicts. Repeat with read-only and revoked tokens, then remove only the smoke resources. Confirm remote MCP uses each caller's token rather than a server-wide token.

[AGENTS_API.md](AGENTS_API.md) defines commands and bounds; [docs/AGENT_CLIENTS.md](docs/AGENT_CLIENTS.md) records official client formats. Test discovery, one read and one authorized write in each actual client account before reporting that client as verified. Grok requires an externally reachable HTTPS endpoint; localhost protocol tests do not establish cloud connectivity.

The backend image packages the CLI. CI is configured to compile Linux/macOS/Windows clients and retain the binaries as artifacts; current run outcomes and published releases must be checked separately.

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

## IDE transport compatibility

The IDE build executes `devmachine/ide/tests/ftp-compatibility.mjs` against its
installed code-server dependency tree on Node 22. It checks actual get-uri FTP
downloads using EPSV/PASV, metadata, cache, missing files and MLSD fallback. It
also checks that an advertised separate data host is refused before a transfer.
This restriction is the default in basic-ftp 6; it is not disabled by the patch.

To repeat the smoke inside a built IDE image:

```sh
docker run --rm --entrypoint node sprintorio/dev-machine-ide:0.1.0 \
  /usr/local/lib/sprintorio-ftp-compatibility.mjs /usr/lib/code-server
```

The script header documents an optional implicit FTPS fixture with a short-lived
certificate. Trust is passed to that client only; there is no global trust change
or certificate-validation bypass. Explicit FTPS and cloud host deployment are
separate checks and are not covered by this smoke.
