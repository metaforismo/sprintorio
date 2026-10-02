<script lang="ts">
	import StandalonePage from '$lib/components/StandalonePage.svelte';
	import { contentModifiedAt, url } from '$lib/config/site';
	import { breadcrumbsFrom } from '$lib/data/routes';
	const meta = { title: 'About Sprintorio — Product Planning and Release Checks', description: 'Sprintorio is a self-hosted workspace for product briefs, milestones, issues and manual release checks. Learn its purpose and current limits.', canonical: url('/about'), modifiedAt: contentModifiedAt('/about') };
	const sections = [
		{ heading: 'A clear path from idea to release', body: 'Sprintorio connects the reason for a project with the work needed to deliver it. Keep the product brief, milestones, issues and manual verification records in the same workspace.', list: ['Product brief: name, objective, success metric and target release', 'Milestones for planning the work', 'Manual test cases with steps, expected outcome, status and evidence', 'Readiness summary to support a human release decision'] },
		{ heading: 'Self-hosted and open source', body: 'The source and deployment configuration are available at github.com/metaforismo/sprintorio under Apache 2.0. Operators manage their own infrastructure, credentials, backups and updates.', list: ['Go API and PostgreSQL storage', 'SvelteKit user interface', 'Docker Compose reference deployment', 'Workspace export/import with uploaded assets', 'No hosted service or support SLA is advertised'] },
		{ heading: 'Current limits', body: 'Sprintorio is an unreleased source build. Manual checks record what a person tested; they do not execute tests or certify a release. A target release is a planning field, not a delivery promise.', list: ['No automatic test execution or deployment approval', 'No enterprise identity workflows', 'Dev Machines are unreleased, optional infrastructure', 'Verify the source and run local checks before production use'] }
	];
</script>
<StandalonePage {meta} heading="About Sprintorio" intro="A self-hosted workspace for turning a product objective into planned work and documented release checks." {sections} breadcrumbs={breadcrumbsFrom('about', 'About')} />
