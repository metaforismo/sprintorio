# Verification record

Verification for the Sprintorio platform revision, 2026-10-02–03. Results describe
this checkout and the named CI runs; they do not certify a hosted deployment.

| Check | Result | Scope |
| --- | --- | --- |
| Repository identity | PASS | Package/module names, public repository links, release metadata, legal notices and all brand assets; fresh public Git history |
| Go race suite | PASS | Full backend with a dedicated PostgreSQL 17 database, including project metadata/null handling, delivery conflicts, workspace/team isolation and atomic rollback |
| Real HTTP persistence | PASS | Registration, immediate login, project creation, saved delivery plans, version conflicts, metadata preservation, invalidation of old test evidence |
| Frontend unit tests | PASS | 16 tests, including current shortcut definitions and accessible workspace recovery |
| Application type checks | PASS | Svelte/TypeScript: zero errors and zero warnings after the final dialog/date fixes |
| Application build/browser suite | PASS | Final UI code: production build and all 33 browser scenarios passed locally and in [CI run 37073664845](https://github.com/metaforismo/sprintorio/actions/runs/37073664845), including disabled development capability, bounded request waits, cycle scope preservation and pending-dialog dismissal |
| Marketing site | PASS | Type checks, production build, SEO for 32 routes and two Chrome responsive/keyboard tests, including 320px overflow checks |
| Runtime helper tests | PASS | Collector, egress and browser proxy race tests under Go 1.25.13 |
| Update script tests | PASS | Self-hosting update script checks |
| npm dependency audits | PASS | UI and WEB each report zero vulnerabilities after compatible dependency updates |
| Go vulnerability scan | PASS | Current vulnerability database, scanner v1.7.0, Go 1.25.13: zero called or imported-package findings; three findings in required modules only |
| Manual Browser verification | PASS | Real local API/PostgreSQL: first team → issue, project creation, delivery brief/milestone/test evidence, reload persistence, cycle creation, issue assignment and activation, keyboard date clearing, failed edit → preserved draft → retry and reload; mobile website/app menus with Escape focus return and 390px project forms without horizontal overflow |
| Docker runtime | PASS on previous PR revision | [Linux CI run 37074314490](https://github.com/metaforismo/sprintorio/actions/runs/37074314490) validated the previous revision; final revision remains subject to its own CI. Docker is unavailable locally |

Automated UI scenarios use mocked API responses and synthetic data. Database,
HTTP and manual Browser checks separately exercise actual persistence. Manual
project test records do not execute automated tests or authorize deployment.

Optional Dev Machines remain disabled by default. Go, OpenSSL, brace-expansion,
js-yaml, tar, ip-address, undici and basic-ftp are patched. The IDE build runs a
compatibility smoke through its actual get-uri 6.0.5 / basic-ftp 6.2.1 dependency
tree. Local Node 22.23.1 checks pass for EPSV/PASV downloads, metadata, cache,
missing files, MDTM fallback to MLSD, and refusal of a separate passive data host.
Implicit FTPS also passes with a fixture CA trusted only by the client; explicit
FTPS was not exercised. Version 6 restricts passive transfers to the control
host; no option re-enables arbitrary data hosts. Final image builds and all
security scans remain visible in [PR #1](https://github.com/metaforismo/sprintorio/pull/1/checks).
Clean application audits are distinct from container and host deployment checks.

The three Go module-only findings concern SSH/OpenPGP packages that the application
does not import or call. This is scanner reachability evidence, not a general
claim that these dependencies are safe for other consumers.

See [TESTING.md](TESTING.md) for reproducible commands and the distinction between
application checks and optional infrastructure checks.

## Agent-workflow revision

The earlier platform evidence above predates this agent-workflow revision. The
following results keep local fixtures, real API checks, client accounts and CI
as separate gates.

| Check | Result | Scope |
| --- | --- | --- |
| CLI/MCP client fixtures | PASS | Seven HTTP/protocol/transfer fixture tests plus a real stdin/stdout child-process test; compact output, write validation, CAS conflicts, auth, limits, redaction and exact Origin checks |
| Operation catalogue | PASS | 145 fixed operations; three initial MCP tool schemas serialize to 879 JSON bytes; detailed operation schemas are requested on demand |
| Client configuration documentation | PASS | Official formats cited for Claude Code, Cursor, Codex, Grok, Muse Code, OpenCode and Gemini CLI; configuration review is not account execution |
| Independent MCP SDK smoke | PASS | Official Node MCP SDK 1.32.0 initialized and called tools over stdio and stateless Streamable HTTP against the actual local API |
| Real API agent workflow | PASS | Synthetic PostgreSQL-backed workspace: MCP project/view creation, CLI issue creation, saved manual evidence, stale-version 409, read-write denial 403, immediate revocation 401, JWT-only token issuance, upload and private ZIP export with overwrite prevention |
| Backend agent revision | PASS | Full Go race suite: 642 passed test executions, six infrastructure/fixture-dependent skips; token and resource isolation, membership changes, migrations and uploads exercised with PostgreSQL 17 |
| UI agent revision | PASS | Final UI production build, 20 unit tests and zero Svelte errors/warnings; all 42 scenarios passed in [CI run 37080585087](https://github.com/metaforismo/sprintorio/actions/runs/37080585087), including collapsed token history, keyboard/IME, retry and mobile flows |
| Manual Browser agent revision | PASS | Real API: saved-context copy feedback, dirty draft protection, project search by team key, client configuration selector, 390px form validation and no horizontal overflow |
| External harness accounts/public HTTP endpoint | NOT RUN | No Claude/Cursor/Codex/Grok/Muse account connection or public deployment is claimed by local fixtures |
| Agent packaging | PASS for CLI compilation | [CI run 37080585087](https://github.com/metaforismo/sprintorio/actions/runs/37080585087) compiled Linux/macOS amd64+arm64 and Windows amd64 clients and uploaded the artifact; release publication remains separate |

Reproduce the client checks using `go test ./internal/agentclient ./cmd/sprintorio`
from `BE`. [AGENTS_API.md](AGENTS_API.md) documents supported operations and limits;
[docs/AGENT_CLIENTS.md](docs/AGENT_CLIENTS.md) separates integration configuration
from tested client-account state. Required redistribution notices remain in
[NOTICE](NOTICE).

The same Tests run passed the Docker lifecycle integration. Local Chrome ran all 42 assertions successfully but its worker teardown did not finish; Linux CI completed normally. This is kept separate from a completed local runner result. Native amd64/arm64 image builds replace QEMU after provider installers failed under emulation; their final CI outcome is checked before merge.
