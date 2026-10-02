<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Plus, Loader2 } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import WorkspaceImportDialog from '$lib/features/workspaces/WorkspaceImportDialog.svelte';
	import { createWorkspace, listWorkspaces } from '$lib/api/workspaces';
	import { appToast } from '$lib/features/toast/toast';
	import { defaultWorkspaceName } from '$lib/utils/default-workspace';
	import { experienceCopy } from '$lib/features/workspaces/experience-copy';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { rememberWorkspace, preferredWorkspace, savedWorkspace } from '$lib/utils/workspace-navigation';
	import { toSlug } from '$lib/utils/slug';
	import { m } from '$lib/paraglide/messages.js';

	let name = $state('');
	let slug = $state('');
	let slugTouched = $state(false);
	let creating = $state(false);
	let ready = $state(false);

	let failed = $state(false);
	const copy = $derived(experienceCopy[getLocale() === 'it' ? 'it' : 'en']);
	async function load() {
		failed = false;
		try {
			await authState.init();
			if (authState.initError) {
				failed = true;
				return;
			}
			if (!authState.authenticated) {
				await goto('/login');
				return;
			}
			const workspaces = await listWorkspaces();
			const workspace = preferredWorkspace(workspaces, savedWorkspace());
			if (workspace) {
				await goto(`/${workspace.slug}/inbox`);
				return;
			}
			name = authState.user?.name ? defaultWorkspaceName(authState.user.name) : '';
			slug = toSlug(authState.user?.name ?? '') || 'workspace';
			ready = true;
		} catch {
			failed = true;
		}
	}
	onMount(load);

	function updateName(value: string) {
		name = value;
		if (!slugTouched) slug = toSlug(value);
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (creating || !name.trim() || !toSlug(slug)) return;
		creating = true;
		try {
			const workspace = await createWorkspace(name.trim(), toSlug(slug));
			rememberWorkspace(workspace.slug);
			await goto(`/${workspace.slug}/inbox`);
		} catch (error: any) {
			appToast.apiError(error, m['workspace_setup.create_failed']());
		} finally {
			creating = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-[var(--color-bg)] px-6 py-12">
	{#if ready}
		<div class="w-full max-w-2xl">
			<div class="text-center">
				<img src="/favicon.svg" alt="Sprintorio" class="mx-auto mb-4 h-12 w-12" />
				<h1 class="text-2xl font-semibold text-[var(--color-text-primary)]">{m['workspace_setup.title']()}</h1>
				<p class="mt-2 text-sm text-[var(--color-text-tertiary)]">{m['workspace_setup.description']()}</p>
			</div>
			<div class="mt-8 grid gap-4 md:grid-cols-2">
				<section class="rounded-lg border border-[var(--app-border)] bg-[var(--color-bg-secondary)] p-5">
					<div class="flex items-center gap-2">
						<Plus size={17} class="text-[var(--app-accent-light)]" />
						<h2 class="font-medium text-[var(--color-text-primary)]">{m['workspace_setup.create_title']()}</h2>
					</div>
					<p class="mt-1 text-xs text-[var(--color-text-tertiary)]">{m['workspace_setup.create_description']()}</p>
					<form onsubmit={create} aria-busy={creating} class="mt-4 space-y-3">
						<label class="block text-sm text-[var(--color-text-secondary)]">
							{m['workspace_transfer.workspace_name']()}
							<input
								value={name}
								oninput={(event) => updateName(event.currentTarget.value)}
								required
								maxlength="100"
								class="mt-1 w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-3 py-2 text-[var(--color-text-primary)] outline-none focus:border-[var(--app-accent)]"
							/>
						</label>
						<label class="block text-sm text-[var(--color-text-secondary)]">
							{m['workspace_transfer.workspace_slug']()}
							<input
								bind:value={slug}
								oninput={() => (slugTouched = true)}
								onblur={() => (slug = toSlug(slug))}
								required
								pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
								maxlength="50"
								class="mt-1 w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-3 py-2 font-mono text-[var(--color-text-primary)] outline-none focus:border-[var(--app-accent)]"
							/>
						</label>
						<Button class="w-full" type="submit" disabled={creating || !name.trim() || !slug.trim()}>
							{#if creating}<Loader2 size={14} class="animate-spin" />{/if}
							{m['workspace_setup.create_button']()}
						</Button>
					</form>
				</section>

				<section class="flex flex-col rounded-lg border border-[var(--app-border)] bg-[var(--color-bg-secondary)] p-5">
					<h2 class="font-medium text-[var(--color-text-primary)]">{m['workspace_setup.import_title']()}</h2>
					<p class="mt-1 flex-1 text-xs text-[var(--color-text-tertiary)]">
						{m['workspace_setup.import_description']()}
					</p>
					<div class="mt-4"><WorkspaceImportDialog /></div>
				</section>
			</div>
		</div>
	{:else if failed}
		<div class="space-y-3 text-center" role="alert">
			<p class="text-sm">{copy.failed}</p>
			<Button onclick={load}>{copy.retry}</Button>
		</div>
	{:else}
		<Loader2 size={20} class="animate-spin text-[var(--color-text-tertiary)]" />
	{/if}
</div>
