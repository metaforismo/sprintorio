# Agent API: CLI and MCP

Sprintorio exposes a fixed workspace operation registry through one Go binary. It calls the same REST API and role checks as the app. There is no arbitrary REST-path tool or terminal command transport.

## Build and credentials

From the repository:

```sh
cd BE
go build -o ../sprintorio ./cmd/sprintorio
```

Set `SPRINTORIO_URL` to the server origin, `SPRINTORIO_WORKSPACE` to the workspace slug and `SPRINTORIO_TOKEN` to a workspace agent token. Inject the token through your harness environment or secret manager. The binary has no token argument, credential file or login command. HTTPS is required except for literal loopback IPs and localhost. Redirects are never followed.

Create/revoke agent tokens in Workspace Settings. Tokens use the `spr_` prefix, expire, and retain the issuing user's current workspace role. `read` permits supported read operations; `write` permits the product workflow; `full` permits workspace administration within that role. Agent tooling cannot create/revoke tokens or change global server settings. A broad token does not upgrade a guest/member to owner. API scope and role failures remain authoritative.

## Discover, inspect, act

```sh
./sprintorio discover
./sprintorio discover issues
./sprintorio schema issues.create
./sprintorio context
./sprintorio call issues.list --input - <<'JSON'
{"query":{"page":1,"per_page":25,"project":"PROJECT_UUID"},"fields":["identifier","title","status"]}
JSON
./sprintorio call issues.create --input - <<'JSON'
{"body":{"title":"Validate release candidate","team_id":"TEAM_UUID","project_id":"PROJECT_UUID","description":"Run the acceptance checks and attach evidence."}}
JSON
```

`--input FILE` also accepts one JSON object. Input has operation path parameters (`id`, `issue`, `team_id`, etc.), optional `query`, optional `body`, and output controls `detail`/`fields`. `issues.get` accepts an issue identifier as `id`; other resource IDs are usually UUIDs. `schema` returns the actual request fields/types, required fields, enums and bounds derived from backend DTOs. Unknown input fields and invalid output fields are rejected before sending a mutation. API validation still checks formats, dates, membership and business rules.

The registry covers issues, subissues, bulk updates/deletes, triage, subscriptions, comments and resolve/reopen, relations, history, projects and delivery plans, teams/statuses/cycles/labels/templates, views, workspace/members, analytics, webhooks, GitHub repository configuration, AI settings, favorites and shared-link management. Optional development-machine operations include creation/lifecycle, checkouts, policy/scope/environment administration, services, terminal-session metadata and bounded agent-run creation/read/cancel/trace. Machine feature flags and backend policy remain enforced. Terminal session creation does not provide a command execution or WebSocket proxy.

An operation is exposed only when the REST API has it. For example, comments have create/list/resolve/reopen; there is no invented comment-edit route, status-get route or relation-update route. Browser-mediated GitHub OAuth/setup, global profile/notifications, WebSockets and ZIP import use the web application. Binary export/upload have the bounded paths below.

`context` returns workspace, teams, labels and projects. Use `statuses.list` with the selected `team_id` to resolve status IDs/categories; use `members.list` for assignees and `cycles.list` for scheduling.

## Compact results and pagination

Default results are compact JSON with stable IDs, names/titles, state and scheduling references. `--detail` includes safe response fields, long descriptions, comment bodies and history values. `--fields id,title,status` selects known output fields. User-defined view filters retain their structure; credential-named keys are excluded. Credential fields are never selectable, and agent token strings are redacted. Shared-link bearer tokens and share URLs are excluded.

Issue lists default to one page of 25 items and cap `per_page` at 100. Returned `page`, `per_page`, `total_count` and `has_more` are retained. Machine event/log pages return an explicit `pagination.next_after_id`, `limit` and `may_have_more`; use `query.after_id` to advance until an empty page. Machine/run lists also accept page/per_page. Agent-run traces use `events_after_id`/`logs_after_id` and `events_limit`/`logs_limit`, retaining `next_event_id`, `next_log_id`, `has_more_events` and `has_more_logs`. Advance each cursor explicitly. Advance issue `query.page` until `has_more` is false; the client never silently stops an all-pages fetch because it never starts one. APIs that return unpaginated arrays return the whole bounded response or an explicit size error. `--jsonl` emits `{"record":...}` lines and a final `{"pagination":...}` envelope for paginated results. No list entries are dropped by projection.

Limits: 2 MiB input per command/MCP line, 8 MiB JSON API response, 20-second HTTP request timeout, no automatic retries. Scope failures, timeouts, conflicts and ambiguous create failures are not automatically retried.

Errors on CLI stderr are JSON with `error.code`, `error.message` and optional HTTP status; stdout remains clean. Exit codes: `0` success, `1` local I/O, `2` input/config/other API rejection, `3` unauthorized/forbidden, `4` conflict, `5` not found, `6` rate limit, `7` network/server failure. Error messages omit credentials and raw API bodies.

## Project delivery and acceptance testing

A project has one versioned delivery plan with product name, objective, success metric, release date, milestones and test cases. Test status is `not_run`, `passed`, `failed` or `blocked`; passed/failed cases require evidence. Milestone status is `planned`, `in_progress` or `done`.

```sh
./sprintorio call delivery.get --input - --detail <<'JSON'
{"id":"PROJECT_UUID"}
JSON
./sprintorio schema delivery.update
./sprintorio call delivery.update --input plan-update.json
```

The update file must contain `{"id":"PROJECT_UUID","body":{"version":LAST_READ_VERSION,"plan":FULL_PLAN}}`. Read with `detail:true` first and preserve unrelated plan fields. Compact delivery responses label their content `plan_summary` and explicitly direct the agent to read full detail before replacement. A stale version returns conflict/exit 4. Read the current version, reconcile the intended change and submit again; never blindly overwrite or auto-retry. Recording a test result does not itself run a test; evidence must describe actual execution.

Use `delivery.summary` for a bounded readout of saved readiness, status counts and the first 20 attention items with stable IDs; `attention_omitted` reports the remainder. The version accompanies the readout. `readiness.ready` requires a complete product brief, at least one test, all tests passed with title/steps/expected result/evidence, and all milestones done. Its basis is `saved_plan_and_test_evidence`; it is a review signal, not release approval or automated execution. Read `delivery.get` with `detail:true` to inspect all records and evidence.

For one case, milestone or a batch, use `delivery.items.update` instead of replacing the full plan:

```sh
./sprintorio schema delivery.items.update
./sprintorio call delivery.summary --input - <<'JSON'
{"id":"PROJECT_UUID"}
JSON
./sprintorio call delivery.items.update --input - <<'JSON'
{"id":"PROJECT_UUID","body":{"version":7,"test_cases":{"upsert":[{"id":"login-smoke","title":"Login opens dashboard","steps":"Log in using the test account","expected_result":"Dashboard appears","status":"passed","evidence":"Smoke run 2026-10-03: dashboard appeared"}]}}}
JSON
```

Each `upsert` replaces the complete item with that ID, or appends a new ID. Omitted collections, unmentioned IDs and product fields are preserved. `remove:["ITEM_ID"]` deletes only known IDs. Each collection permits 200 upserts and 200 removals; the resulting plan remains limited to 200 milestones and 200 tests. Duplicate IDs within a batch, overlapping upsert/remove IDs, unknown removals and empty changes fail without saving. Omitted or null collections do not clear existing records. Batches commit atomically under the same version check and `project:manage` permission as full updates. Edits to a test definition reset an unchanged old outcome/evidence to `not_run`; supply fresh evidence only after an actual new run. Stale versions return 409/exit 4 with no automatic retry.


## Files

```sh
./sprintorio upload --file proof.txt
./sprintorio export --output workspace.zip
```

CLI uploads use authenticated multipart requests, max 10 MiB, with backend file type checks. Export streams a ZIP archive, max 64 MiB, to a newly created file with mode 0600; existing files are not replaced and failed downloads remove their partial file. The server excludes credentials from workspace exports; local command credentials are never saved.

For MCP, `assets.upload` accepts `body.filename` plus base64 `body.base64`, max 1 MiB decoded and within the 2 MiB message limit. `assets.get` and `workspace.export` have `kind: authenticated_download`: their action returns the fixed API URL and authentication requirement instead of binary content. Download with the caller's own authenticated HTTP client, or use CLI export. The MCP server does not write remote callers' files on its host. ZIP import uses the web transfer flow with preview/reconfiguration checks.

## Local stdio MCP

Launch `sprintorio mcp` with the three environment variables. Point a stdio-compatible harness at the binary and argument `mcp`; use that harness's environment/secret configuration to inject credentials. For clients with a `mcpServers` JSON configuration:

```json
{"mcpServers":{"sprintorio":{"command":"/absolute/path/sprintorio","args":["mcp"]}}}
```

Ensure the process inherits the credentials; do not paste them into a committed config. Client-specific secret injection and config locations vary.

Only three tools are registered:

- `discover {resource?}` returns compact operation metadata.
- `schema {operation}` returns one exact argument schema on demand.
- `action {operation,input}` executes that operation, with the same output options as CLI.

The measured initial tool catalogue is 879 JSON bytes for three tools (see `TestMCPStdinLifecycleAndActions`); operation schemas are not loaded until requested. Byte size is reproducible; exact model token counts depend on tokenizer/harness and are not claimed. Successful results appear once in `structuredContent.result` with empty text content. Tool execution errors set `isError:true`, a short error text and `structuredContent.error`. Protocol errors use JSON-RPC error codes. Standard JSON-RPC initialization, protocol negotiation, initialized notifications, ping, tools/list and tools/call use newline-delimited messages. Diagnostics never go to stdout. Supported versions are 2025-11-25 and 2025-06-18; unsupported versions negotiate the latest supported version.

## Remote Streamable HTTP MCP

```sh
SPRINTORIO_URL=https://sprintorio.example \
SPRINTORIO_WORKSPACE=demo \
./sprintorio mcp-http --listen 127.0.0.1:8091
```

Endpoint: `http://127.0.0.1:8091/mcp`. The default listener is loopback. For a remote deployment, explicitly configure a listener and place HTTPS termination/access controls in front of it. No tunnel or public deployment is created by this command.

Each HTTP request requires `Authorization: Bearer AGENT_TOKEN`; the caller token is checked against the workspace API even for initialize and tool discovery. Actions forward that same caller token. The process never uses a shared server token for remote callers. Native clients without an Origin header work; browser Origins require an exact `--origins https://trusted-client.example` allowlist. Wildcards are rejected. POST returns JSON, notifications return 202, GET/DELETE return 405. The stateless transport has no session IDs, SSE streams, server-initiated requests or resumability.

Configure remote-only clients, including a Grok remote-MCP integration, with your deployed HTTPS `/mcp` endpoint and Bearer header using the client's supported secret/header mechanism. This repository supplies the endpoint implementation; an actual public HTTPS deployment and client account smoke test are separate checks. No unverified configuration format for Muse or another client is implied.

Protocol references: [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle), [stdio and Streamable HTTP](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports), [MCP tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools). The implementation uses the Go standard library; it adds no MCP SDK dependency.
