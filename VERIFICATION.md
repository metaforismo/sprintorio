# Verification record

Local verification for the initial Sprintorio distribution, 2026-10-02. This is a
record of checks performed on this checkout, not a deployment certification.

| Check | Result | Scope |
| --- | --- | --- |
| Repository identity | PASS | Package names, module path, repository links, logos, release metadata, required legal notices |
| Go race and coverage suite | PASS | Full backend suite with a dedicated PostgreSQL 17 database; 31.6% statement coverage |
| Real HTTP persistence | PASS | Registration, immediate login, project creation, saved delivery plans, version conflicts, metadata preservation, invalidation of old test evidence |
| Frontend unit tests | PASS | 14 tests |
| Application type checks | PASS | No errors; one existing shortcut initialization warning |
| Application production build | PASS | Static output generated; existing shortcut initialization and large-bundle warnings |
| Application browser suite | PASS | All 25 tests against the production preview in Chrome; includes delivery persistence/readiness, conflict recovery, guest permissions, and mobile overflow |
| Marketing site | PASS | Type checks, production build, SEO checks for 32 routes, and responsive browser checks at 320px |
| Runtime helper tests | PASS | Collector, egress, and browser proxy Go race tests |
| Update script tests | PASS | Self-hosting update script checks |
| Docker runtime | NOT RUN | Docker is unavailable on the local host; container integration has a separate CI job |

Browser scenarios use mocked API responses and sample data. PostgreSQL and HTTP
checks separately verify actual persistence. Manual project test records do not
execute automated tests. See [TESTING.md](TESTING.md) to reproduce the checks.
