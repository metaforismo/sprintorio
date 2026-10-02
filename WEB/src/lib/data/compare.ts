/**
 * Content registry for comparison pages.
 * Each entry maps to /compare/[slug].
 */
import type { ContentRegistry } from './routes';

export const compare: ContentRegistry = {
	'sprintorio-vs-linear': {
		slug: 'sprintorio-vs-linear',
		title: 'Sprintorio vs Linear — Comparison',
		description:
			'Compare Sprintorio and Linear by hosting model, license, issue workflow, planning, integrations, product maturity, and software cost.',
		heading: 'Sprintorio vs Linear',
		intro:
			'Both products emphasize fast, keyboard-oriented issue tracking. The main decision is not superficial feature parity: Linear is a mature hosted product, while Sprintorio is an early self-hosted project with complete source access and no software license fee.',
		sections: [
			{
				heading: 'Hosting and source access',
				body: 'The products use different delivery models.',
				list: [
					'Linear is a proprietary hosted service and does not offer a self-hosted edition.',
					'Sprintorio is self-hosted from one public Apache 2.0 repository. There is no hosted Sprintorio service.'
				]
			},
			{
				heading: 'Core issue tracking',
				body: 'Sprintorio covers a smaller product surface but differs on issue ownership.',
				list: [
					'Linear provides a mature issue workflow, command menu, filters and keyboard shortcuts.',
					'Sprintorio provides multiple assignees per issue, sub-issues, relations, labels, comments, saved views and team-specific statuses.'
				]
			},
			{
				heading: 'Planning: cycles and projects',
				body: 'Both products support time-boxed work and project grouping, but Linear’s planning layer is broader.',
				list: [
					'Linear: cycles, projects, roadmaps, and initiatives for grouping work across teams.',
					'Sprintorio has cycles with burndown and velocity charts, projects with progress and a Gantt view, and workspace/team analytics with configurable insights. It has no initiatives layer.'
				]
			},
			{
				heading: 'Integrations and automation',
				body: 'Check required integrations before choosing either product.',
				list: [
					'Linear: documents integrations for GitHub, GitLab, Slack, its API, and enterprise identity features.',
					'Sprintorio integrates with GitHub and generic webhooks. Its development branch includes an unreleased opt-in Dev Machines subsystem for agentic coding runs. It does not currently provide GitLab, Slack, SSO, SCIM or LDAP integrations.'
				]
			},
			{
				heading: 'Cost model',
				body: 'Published prices and plan limits can change; the linked pricing page is authoritative.',
				list: [
					'Linear offers a limited free plan and paid per-user plans. Hosting and product operations are included in that service.',
					'Sprintorio charges no software license fee. You pay for infrastructure and operate backups, monitoring and updates.'
				]
			},
			{
				heading: 'How to choose',
				body: 'Choose based on operating model and required maturity, not on headline similarity.',
				list: [
					'Choose Linear when a managed service, broader integrations and a mature product are more important than self-hosting.',
					'Choose Sprintorio when self-hosting, multiple assignees, source access and no software license fee are hard requirements—and the current MVP covers your workflow.'
				]
			},
			{
				heading: 'Sources',
				body: 'Review the first-party sources for the latest product details.',
				links: [
					{ label: 'Linear pricing', href: 'https://linear.app/pricing' },
					{ label: 'Linear documentation', href: 'https://linear.app/docs' },
					{ label: 'Sprintorio source and documentation', href: 'https://github.com/metaforismo/sprintorio' }
				]
			}
		],
		footnotes:
			'These pages describe product tradeoffs. External feature and pricing details require verification against the linked product documentation before purchasing. Sprintorio is an independent project and is not affiliated with Linear.'
	},

	'sprintorio-vs-plane': {
		slug: 'sprintorio-vs-plane',
		title: 'Sprintorio vs Plane — Comparison',
		description:
			'Compare Sprintorio and Plane by license, self-hosting, product scope, planning features, integrations, feature tiers, and operational fit.',
		heading: 'Sprintorio vs Plane',
		intro:
			'Both products can run on your infrastructure, but they solve different-sized problems. Plane is a broad project-management suite with commercial plans; Sprintorio is a focused issue tracker distributed as one Apache 2.0 edition.',
		sections: [
			{
				heading: 'License and editions',
				body: 'Both projects publish source, but their licenses and edition models differ.',
				list: [
					'Plane’s public repository uses AGPL-3.0. Plane offers cloud and self-hosted products with paid feature plans.',
					'Sprintorio uses Apache 2.0 and ships one self-hosted edition. It has no paid feature plan or enterprise repository.'
				]
			},
			{
				heading: 'Scope and focus',
				body: 'Plane covers more workflows; Sprintorio deliberately covers fewer.',
				list: [
					'Plane includes work items, cycles, modules, initiatives, pages, wiki, intake, dashboards and multiple layouts.',
					'Sprintorio focuses on issues, cycles, projects, saved views, analytics insights, public sharing and GitHub activity. It has no wiki, modules or customizable dashboards.'
				]
			},
			{
				heading: 'Core issue tracking',
				body: 'Both support structured work, but their ownership and customization models differ.',
				list: [
					'Plane: documents multiple work-item views, properties, and project-management workflows.',
					'Sprintorio includes multiple assignees, sub-issues, relations, hierarchical labels, issue templates, favorites and read-only public links.'
				]
			},
			{
				heading: 'Planning and visualization',
				body: 'Plane offers more layouts and planning layers. Sprintorio keeps planning closer to issues.',
				list: [
					'Plane: cycles, modules, multiple layout views (list, board, Gantt, calendar).',
					'Sprintorio: cycles with burndown/velocity, projects with Gantt view, list and board issue views, and a built-in insights page with burn-up trends.'
				]
			},
			{
				heading: 'Integrations',
				body: 'Plane documents a wider integration and migration catalog.',
				list: [
					'Plane: GitHub, GitLab, Slack integrations. Importer for multiple platforms. API and webhooks.',
					'Sprintorio: self-configuring GitHub App with auto-linking, auto-transitions, and real-time WebSocket events. Webhooks. Unreleased opt-in Dev Machines development. Workspace export/import includes uploaded assets and regenerates entity IDs.'
				]
			},
			{
				heading: 'How to choose',
				body: 'The relevant tradeoff is breadth versus a smaller, uniform edition.',
				list: [
					'Choose Plane when pages, wiki, modules, dashboards, importers or its wider integration catalog are required.',
					'Choose Sprintorio when a focused tracker, multiple assignees, Apache 2.0, and one ungated self-hosted edition matter more than breadth.'
				]
			},
			{
				heading: 'Sources',
				body: 'Review the first-party sources for the latest edition, licensing, and feature details.',
				links: [
					{ label: 'Plane documentation', href: 'https://docs.plane.so/' },
					{ label: 'Plane source repository', href: 'https://github.com/makeplane/plane' },
					{ label: 'Sprintorio source and documentation', href: 'https://github.com/metaforismo/sprintorio' }
				]
			}
		],
		footnotes:
			'External feature, edition and licensing details must be checked against the linked first-party documentation before making a purchasing decision. Sprintorio is not affiliated with Plane.'
	}
};
