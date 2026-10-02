<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { listWorkspaces } from '$lib/api/workspaces';
	import { preferredWorkspace, savedWorkspace } from '$lib/utils/workspace-navigation';
	import { experienceCopy } from '$lib/features/workspaces/experience-copy';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { Button } from '$lib/components/ui/button';
	import { Loader2 } from 'lucide-svelte';
	let failed = $state(false);
	const copy = $derived(experienceCopy[getLocale() === 'it' ? 'it' : 'en']);
	async function enter() {
		failed = false;
		await authState.init();
		if (authState.initError) {
			failed = true;
			return;
		}
		if (!authState.authenticated) {
			await goto('/login');
			return;
		}
		try {
			const workspace = preferredWorkspace(await listWorkspaces(), savedWorkspace());
			await goto(workspace ? `/${workspace.slug}/inbox` : '/workspace-setup');
		} catch {
			failed = true;
		}
	}
	onMount(enter);
</script>

<div class="flex min-h-dvh flex-col items-center justify-center gap-3" role="status">
	{#if failed}<p class="text-sm text-[var(--color-text-secondary)]">{copy.failed}</p>
		<Button onclick={enter}>{copy.retry}</Button>
	{:else}<Loader2 size={20} class="animate-spin text-[var(--color-text-tertiary)]" /><span class="sr-only"
			>{copy.loading}</span
		>{/if}
</div>
