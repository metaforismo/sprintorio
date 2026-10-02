<script lang="ts">
	import { page } from '$app/state';

	const isHome = $derived(page.url.pathname === '/');

	const productLinks = $derived([
		{ label: 'Open source', homeHref: '#open-source', routeHref: '/open-source' },
		{ label: 'Features', homeHref: '#features', routeHref: '/features' },
		{ label: 'Pricing', homeHref: '#pricing', routeHref: '/#pricing' },
		{ label: 'Self-host', homeHref: '#deploy', routeHref: '/self-hosting' },
		{ label: 'Compare', homeHref: '/compare', routeHref: '/compare' },
		{ label: 'Alternatives', homeHref: '/alternatives', routeHref: '/alternatives' }
	]);

	function linkHref(link: { homeHref: string; routeHref: string }): string {
		return isHome ? link.homeHref : link.routeHref;
	}

</script>

<footer class="border-t border-border bg-black/30">
	<div class="mx-auto max-w-6xl px-6 py-12">
		<div class="grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
			<div class="flex flex-col gap-3 sm:col-span-2">
				<img src="/logo_white.svg" alt="Sprintorio" class="h-6 w-auto self-start" width={96} height={24} />
				<p class="text-xs text-muted-foreground">
					Product briefs, milestones, issues and manual release checks. Self-hosted and open source.
				</p>
			</div>

			<div class="flex flex-col gap-3 text-sm">
				<p class="text-xs font-semibold tracking-widest text-muted-foreground uppercase">Product</p>
				{#each productLinks as link}
					<a href={linkHref(link)} class="text-muted-foreground transition-colors hover:text-foreground">{link.label}</a>
				{/each}
			</div>

			<div class="flex flex-col gap-3 text-sm">
				<p class="text-xs font-semibold tracking-widest text-muted-foreground uppercase">Project</p>
				<a
					href="https://github.com/metaforismo/sprintorio"
					target="_blank"
					rel="noopener"
					class="text-muted-foreground transition-colors hover:text-foreground">GitHub</a
				>
			<a
				href="/license"
				class="text-muted-foreground transition-colors hover:text-foreground">License</a
			>
				<a href="/privacy" class="text-muted-foreground transition-colors hover:text-foreground"
					>Privacy</a
				>
				<a href="/security" class="text-muted-foreground transition-colors hover:text-foreground">Security</a>
				<a href="/about" class="text-muted-foreground transition-colors hover:text-foreground">About</a>
				<a href="/roadmap" class="text-muted-foreground transition-colors hover:text-foreground">Roadmap</a>
			</div>
		</div>

		<div
			class="mt-10 flex flex-col items-center justify-between gap-3 border-t border-border pt-6 sm:flex-row"
		>
			<p class="text-xs text-muted-foreground">
				Sprintorio · Apache 2.0 license.
			</p>
		</div>
	</div>
</footer>
