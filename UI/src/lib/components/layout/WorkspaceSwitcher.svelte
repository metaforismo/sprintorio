<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Workspace } from '$lib/types/workspace';
	import { createWorkspace, listWorkspaces } from '$lib/api/workspaces';
	import * as Popover from '$lib/components/ui/popover';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Plus, ChevronsUpDown, Check, Loader2 } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { appToast } from '$lib/features/toast/toast';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, setLocale } from '$lib/paraglide/runtime.js';
	import { rememberWorkspace } from '$lib/utils/workspace-navigation';
	import { experienceCopy } from '$lib/features/workspaces/experience-copy';
	import { toSlug } from '$lib/utils/slug';

	let {
		currentWorkspace,
		slug
	}: {
		currentWorkspace: Workspace;
		slug: string;
	} = $props();

	let open = $state(false);
	let showCreateWorkspace = $state(false);
	let workspaces = $state<Workspace[]>([]);
	let newWorkspaceName = $state('');
	let newWorkspaceSlug = $state('');
	let slugEdited = $state(false);
	let creating = $state(false);
	let failed = $state(false);
	let loading = $state(false);
	const copy = $derived(experienceCopy[getLocale() === 'it' ? 'it' : 'en']);

	async function load() {
		loading = true;
		failed = false;
		try {
			workspaces = await listWorkspaces();
		} catch {
			failed = true;
		} finally {
			loading = false;
		}
	}
	onMount(load);

	function switchWorkspace(ws: Workspace) {
		open = false;
		if (ws.slug !== slug) {
			rememberWorkspace(ws.slug);
			goto(`/${ws.slug}/my-issues`);
		}
	}

	function handleNameInput() {
		if (!slugEdited) {
			newWorkspaceSlug = toSlug(newWorkspaceName);
		}
	}

	function openCreateWorkspace() {
		open = false;
		showCreateWorkspace = true;
	}

	async function handleCreateWorkspace(e: Event) {
		e.preventDefault();
		if (creating) return;
		const workspaceName = newWorkspaceName.trim();
		const workspaceSlug = toSlug(newWorkspaceSlug || newWorkspaceName);
		if (!workspaceName || !workspaceSlug) return;

		creating = true;
		try {
			const workspace = await createWorkspace(workspaceName, workspaceSlug);
			workspaces = [...workspaces, workspace].sort((a, b) => a.name.localeCompare(b.name));
			rememberWorkspace(workspace.slug);
			appToast.success(m['sidebar.workspace_created']());
			showCreateWorkspace = false;
			newWorkspaceName = '';
			newWorkspaceSlug = '';
			slugEdited = false;
			goto(`/${workspace.slug}/inbox`);
		} catch (err: any) {
			appToast.apiError(err, m['sidebar.failed_create_workspace']());
		} finally {
			creating = false;
		}
	}
</script>

<Popover.Root bind:open>
	<Popover.Trigger
		aria-label={m['sidebar.workspaces']()}
		class="flex w-full min-w-0 items-center gap-2 rounded-md px-1 py-0.5 hover:bg-[var(--color-bg-hover)] max-md:min-h-11"
	>
		<div
			class="flex h-6 w-6 shrink-0 items-center justify-center rounded bg-[var(--app-accent)] text-xs font-bold text-[var(--app-accent-foreground)]"
		>
			{currentWorkspace.name.charAt(0).toUpperCase()}
		</div>
		<span class="flex-1 truncate text-left text-sm font-medium text-[var(--color-text-primary)]">
			{currentWorkspace.name}
		</span>
		<ChevronsUpDown size={14} class="shrink-0 text-[var(--color-text-tertiary)]" />
	</Popover.Trigger>
	<Popover.Content class="w-56 p-1" align="start">
		<div class="px-2 py-1">
			<span class="text-[10px] font-medium uppercase text-[var(--color-text-tertiary)]"
				>{m['sidebar.workspaces']()}</span
			>
		</div>
		{#if failed}<div class="p-2 text-xs" role="alert">
				<p>{copy.failed}</p>
				<Button variant="outline" size="sm" class="mt-2" onclick={load}>{copy.retry}</Button>
			</div>{:else if loading}<p class="p-2 text-xs" role="status">{copy.loading}</p>{/if}
		{#each workspaces as ws}
			<button
				onclick={() => switchWorkspace(ws)}
				class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]"
			>
				<div
					class="flex h-5 w-5 items-center justify-center rounded bg-[var(--app-accent)] text-[9px] font-bold text-[var(--app-accent-foreground)]"
				>
					{ws.name.charAt(0).toUpperCase()}
				</div>
				<span class="flex-1 truncate text-left">{ws.name}</span>
				{#if ws.slug === slug}
					<Check size={14} class="text-[var(--app-accent)]" />
				{/if}
			</button>
		{/each}
		<div class="mt-1 border-t border-[var(--app-border)] pt-1">
			<button
				onclick={openCreateWorkspace}
				class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]"
			>
				<Plus size={14} />
				{m['sidebar.create_workspace']()}
			</button>
		</div>
	</Popover.Content>
</Popover.Root>

<Dialog.Root bind:open={showCreateWorkspace}>
	<Dialog.Content class="sm:max-w-md border-[var(--app-border)] bg-[var(--color-bg-secondary)]">
		<Dialog.Header>
			<Dialog.Title>{m['sidebar.create_workspace']()}</Dialog.Title>
			<Dialog.Description>{m['sidebar.create_workspace_desc']()}</Dialog.Description>
		</Dialog.Header>

		<form onsubmit={handleCreateWorkspace} class="space-y-4 py-2">
			<div>
				<label for="workspace-name" class="mb-1 block text-sm text-[var(--color-text-secondary)]"
					>{m['sidebar.workspace_name']()}</label
				>
				<input
					id="workspace-name"
					type="text"
					bind:value={newWorkspaceName}
					oninput={handleNameInput}
					required
					maxlength="100"
					placeholder={m['sidebar.workspace_name_placeholder']()}
					class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[var(--app-accent)]"
				/>
			</div>

			<div>
				<label for="workspace-slug" class="mb-1 block text-sm text-[var(--color-text-secondary)]"
					>{m['sidebar.workspace_url']()}</label
				>
				<input
					id="workspace-slug"
					type="text"
					bind:value={newWorkspaceSlug}
					oninput={() => (slugEdited = true)}
					onblur={() => (newWorkspaceSlug = toSlug(newWorkspaceSlug))}
					required
					maxlength="50"
					placeholder={m['sidebar.workspace_url_placeholder']()}
					class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[var(--app-accent)]"
				/>
				<p class="mt-1 text-xs text-[var(--color-text-tertiary)]">{m['sidebar.workspace_url_desc']()}</p>
			</div>

			<Dialog.Footer>
				<Button type="button" variant="outline" onclick={() => (showCreateWorkspace = false)} disabled={creating}
					>{m['sidebar.cancel']()}</Button
				>
				<Button type="submit" disabled={creating || !newWorkspaceName.trim() || !newWorkspaceSlug.trim()}>
					{#if creating}
						<Loader2 size={14} class="animate-spin" />
						{m['sidebar.creating']()}
					{:else}
						{m['sidebar.create_workspace']()}
					{/if}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
