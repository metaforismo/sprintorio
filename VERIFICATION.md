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
| Docker runtime | PASS on previous PR revision | [Linux CI run 37071976260](https://github.com/metaforismo/sprintorio/actions/runs/37071976260) validated the previous revision; final revision remains subject to its own CI. Docker is unavailable locally |

Automated UI scenarios use mocked API responses and synthetic data. Database,
HTTP and manual Browser checks separately exercise actual persistence. Manual
project test records do not execute automated tests or authorize deployment.

Optional Dev Machines remain disabled by default. Compatible fixes cover Go, OpenSSL,
brace-expansion, js-yaml, tar, ip-address and undici. The IDE still includes
basic-ftp 5.3.1 through get-uri: CVE-2026-102990 requires the incompatible 6.2.1
upgrade and has no compatible parent update at verification time. Final image
builds/scans are tracked in PR #1. Do not infer container security from clean
application dependency audits or enable this subsystem as a verified deployment.

The three Go module-only findings concern SSH/OpenPGP packages that the application
does not import or call. This is scanner reachability evidence, not a general
claim that these dependencies are safe for other consumers.

See [TESTING.md](TESTING.md) for reproducible commands and the distinction between
application checks and optional infrastructure checks.
