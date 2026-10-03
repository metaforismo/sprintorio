# Changelog

## Unreleased

- Compact delivery operations: `delivery.summary` and versioned `delivery.items.update`; the agent catalogue now contains 147 operations.
- Delivery sections use tabs, compact manual-test rows and editing dialogs; shared help popovers and explicit history toggles replace native disclosures.
- Agent connection tabs, a shared client selector, installation dialog and stable clipboard feedback; release notes remain directly readable.
- Project views follow URL navigation and retain unsaved delivery drafts; project action menus use shared accessible controls.
- Context-copy actions report success or failure through a toast without shifting the action layout.

- Build optional runtime images on native amd64/arm64 runners with architecture-specific caches.
- Saved delivery context, accessible command search with stale-result protection, and collapsed inactive-token history.
- Accept valid UTF-8 text, Markdown and log attachments with detected charset parameters; retain MIME and extension checks.

- Workspace agent tokens with read/write/full scopes, expiry, revocation and current-role enforcement; Settings → Agents provides setup examples.
- One Go client for CLI, local stdio MCP and per-caller authenticated Streamable HTTP MCP, with 147 fixed operations and three compact discovery/schema/action tools.
- Bounded JSON/file transfers, safe field projection, explicit pagination, structured errors and delivery-plan version checks across agent workflows.
- Backend image packaging and CI client-artifact build configuration; official-format integration guides for common agent harnesses. Publishing and current CI results remain separate checks.

- Approved A1 brand identity: clean SVG sources, favicon, wordmarks, and social card.
- First-workspace guide, remembered accessible workspace, and draft-preserving team/issue/cycle/project creation.
- Keyboard-accessible selectors with a single composed trigger, visible focus, and reduced-motion button feedback.
- Acceptance, regression, and accessibility test starters; searchable manual test rows and dedicated editing dialogs.
- Project metadata validation, nullable date/association updates, scoped team resources, and completed-only progress.
- Atomic team/status provisioning, checked references, and rollback on failed writes.
- Lazy custom team icons, batched project progress, and on-demand timeline/development data.
- Retryable timeline loading, partial-chart notices, stale-response guards, and bounded session recovery.
- Optional Dev Machines navigation follows the server capability; member management accurately describes existing-account access.
- Keyboard-accessible date clearing, labelled cycle-edit fields and preserved scope after metadata edits.
- Compatible frontend dependency patches, Go 1.25.13, OpenSSL and Echo/network security updates.
- Tested basic-ftp 6 upgrade in the optional IDE; passive data hosts are restricted and the build verifies actual get-uri transfers.
- Current desktop/mobile product screenshots in the README and website.
- CI cancels superseded runs and uses a Go-compatible vulnerability scanner.
- Standalone Sprintorio identity, new repository links, logos, and documentation.
- Project delivery plans: product briefs, milestones, and manual test cases.
- Versioned saves prevent conflicting edits from silently overwriting one another.
- Delivery plans travel with workspace exports and imports.
- Repeatable frontend, backend, browser, and branding checks.
- Fresh-instance Docker startup applies migrations and correctly routes UI/API traffic.
- Refresh tokens have unique IDs so immediate logins and rotations do not collide.
