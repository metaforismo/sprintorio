<script lang="ts">
	import StandalonePage from '$lib/components/StandalonePage.svelte';
	import { contentModifiedAt, url } from '$lib/config/site';
	import { breadcrumbsFrom } from '$lib/data/routes';
	const meta = { title: 'Sprintorio Privacy — Website and Self-Hosted Data', description: 'Understand the data boundaries of the Sprintorio static website and self-hosted application, and the responsibilities of each instance operator.', canonical: url('/privacy'), modifiedAt: contentModifiedAt('/privacy') };
	const sections = [
		{ heading: 'This website', body: 'This is a static product and documentation site. Its source does not include analytics tracking or a contact form. The hosting operator may process connection metadata such as IP addresses and request logs; operators should publish their own contact details and retention policy for their deployment.' },
		{ heading: 'Browser preferences and external requests', body: 'A cookie preference can be stored locally in your browser. The site may request the public release manifest from GitHub to display release information. Following repository and documentation links sends you to external services governed by their own privacy policies.' },
		{ heading: 'Self-hosted application data', body: 'Each instance operator controls the application database and upload storage. Accounts, project briefs, milestones, issues, comments and manual test evidence belong to that instance. Avoid placing unnecessary personal information or credentials in descriptions and evidence.' },
		{ heading: 'Operator responsibilities', body: 'Before offering an instance to other people, identify its responsible operator and publish a privacy notice covering purposes, applicable legal basis, access, retention, backups, subprocessors and a contact channel. Review enabled integrations and optional infrastructure separately.' },
		{ heading: 'Requests about your data', body: 'Contact the operator of the instance you use for access, correction or deletion requests. This source repository does not operate a shared hosted service and cannot inspect data stored in independently hosted instances.' }
	];
</script>
<StandalonePage {meta} heading="Privacy" intro="Data handling depends on who hosts the website and application. This page describes the source-level boundaries." {sections} breadcrumbs={breadcrumbsFrom('privacy', 'Privacy')} />
