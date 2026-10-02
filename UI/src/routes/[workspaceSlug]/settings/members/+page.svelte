<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getWorkspace } from '$lib/api/workspaces';
	import { experienceCopy } from '$lib/features/workspaces/experience-copy';
	import { Loader2 } from 'lucide-svelte';
	import type { WorkspaceMember } from '$lib/types/workspace';
	import { listMembers, updateMemberRole, removeMember, inviteMember } from '$lib/api/members';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import * as Popover from '$lib/components/ui/popover';
	import { appToast } from '$lib/features/toast/toast';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, setLocale } from '$lib/paraglide/runtime.js';
	import { UserPlus, Trash2 } from 'lucide-svelte';

	const slug = $derived(page.params.workspaceSlug ?? '');

	let members = $state<WorkspaceMember[]>([]);
	let loading = $state(true);
	let showInvite = $state(false);
	let inviteEmail = $state('');
	let inviteRole = $state('member');
	let adding = $state(false);
	let loadError = $state(false);
	let canManage = $state(false);
	const copy = $derived(experienceCopy[getLocale() === 'it' ? 'it' : 'en']);

	const roles = ['owner', 'admin', 'member', 'guest'];

	async function load() {
		loading = true;
		loadError = false;
		try {
			const [result, workspace] = await Promise.all([listMembers(slug), getWorkspace(slug)]);
			members = result;
			canManage = ['owner', 'admin'].includes(workspace.current_user_role);
		} catch {
			loadError = true;
		} finally {
			loading = false;
		}
	}
	onMount(load);

	async function handleRoleChange(userId: string, role: string) {
		try {
			await updateMemberRole(slug, userId, role);
			members = members.map((m) => (m.user_id === userId ? { ...m, role } : m));
			appToast.success(m['settings.members.role_updated']());
		} catch (err: any) {
			appToast.apiError(err, m['settings.members.failed_update_role']());
		}
	}

	async function handleRemove(userId: string) {
		try {
			await removeMember(slug, userId);
			members = members.filter((m) => m.user_id !== userId);
			appToast.success(m['settings.members.removed']());
		} catch (err: any) {
			appToast.apiError(err, m['settings.members.failed_remove']());
		}
	}

	async function handleInvite(event: SubmitEvent) {
		event.preventDefault();
		if (adding || !inviteEmail.trim() || !canManage) return;
		adding = true;
		try {
			await inviteMember(slug, inviteEmail.trim(), inviteRole);
			appToast.success(copy.memberAdded);
			showInvite = false;
			inviteEmail = '';
			inviteRole = 'member';
			members = await listMembers(slug);
		} catch (err: any) {
			appToast.apiError(err, copy.memberFailed);
		} finally {
			adding = false;
		}
	}
</script>

<div class="mx-auto max-w-2xl px-4 py-6 sm:px-8 sm:py-10">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold text-[var(--color-text-primary)]">{m['settings.members.title']()}</h1>
		{#if canManage}<button
				onclick={() => (showInvite = true)}
				class="flex items-center gap-1.5 rounded-md bg-[var(--app-accent)] px-3 py-1.5 text-sm text-[var(--app-accent-foreground)] hover:bg-[var(--app-accent-hover)]"
			>
				<UserPlus size={14} />
				{copy.addMember}
			</button>{/if}
	</div>

	<div class="mt-8">
		{#if loading}
			<div role="status" class="flex items-center gap-2 text-sm">
				<Loader2 size={16} class="animate-spin" /><span class="sr-only">{copy.loading}</span>
			</div>
		{:else if loadError}<div role="alert" class="space-y-3">
				<p class="text-sm">{copy.failed}</p>
				<Button onclick={load}>{copy.retry}</Button>
			</div>
		{:else if members.length === 0}
			<p class="text-sm text-[var(--color-text-secondary)]">{m['settings.members.no_members']()}</p>
		{:else}
			<div class="overflow-hidden rounded-lg border border-[var(--app-border)] bg-[var(--color-bg-secondary)]">
				{#each members as member, i}
					<div
						class="flex flex-wrap items-center justify-between gap-3 px-4 py-3.5 {i > 0
							? 'border-t border-[var(--app-border)]'
							: ''}"
					>
						<div class="flex items-center gap-3">
							<div
								class="flex h-8 w-8 items-center justify-center rounded-full bg-[var(--app-accent)] text-xs font-medium text-[var(--app-accent-foreground)]"
							>
								{(member.name || member.email).charAt(0).toUpperCase()}
							</div>
							<div>
								<p class="text-sm font-medium text-[var(--color-text-primary)]">
									{member.name || m['settings.members.unnamed']()}
								</p>
								<p class="break-all text-xs text-[var(--color-text-tertiary)]">{member.email}</p>
							</div>
						</div>
						{#if canManage}<div class="flex items-center gap-2">
								<Popover.Root>
									<Popover.Trigger
										class="rounded-md border border-[var(--app-border)] px-2.5 py-1 text-xs capitalize text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]"
									>
										{(m as unknown as Record<string, () => string>)['common.role.' + member.role]()}
									</Popover.Trigger>
									<Popover.Content class="w-36 p-1" align="end">
										{#each roles as role}
											<button
												onclick={() => handleRoleChange(member.user_id, role)}
												class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm capitalize text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] {member.role ===
												role
													? 'bg-[var(--color-bg-hover)]'
													: ''}"
											>
												{(m as unknown as Record<string, () => string>)['common.role.' + role]()}
											</button>
										{/each}
									</Popover.Content>
								</Popover.Root>
								<button
									onclick={() => handleRemove(member.user_id)}
									class="rounded p-1 text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)] hover:text-red-500"
									title={m['settings.members.remove']()}
								>
									<Trash2 size={14} />
								</button>
							</div>{:else}<span class="text-xs capitalize text-[var(--color-text-secondary)]">{member.role}</span>{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- Invite Dialog -->
<Dialog.Root bind:open={showInvite}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>{copy.addMember}</Dialog.Title>
			<Dialog.Description>{copy.memberHint}</Dialog.Description>
		</Dialog.Header>
		<form onsubmit={handleInvite} aria-busy={adding} class="space-y-4 py-4">
			<div>
				<label for="invite-email" class="mb-1 block text-sm text-[var(--color-text-secondary)]"
					>{m['settings.members.email']()}</label
				>
				<input
					id="invite-email"
					type="email"
					required
					autocomplete="email"
					disabled={adding}
					bind:value={inviteEmail}
					placeholder={m['settings.members.email_placeholder']()}
					class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[var(--app-accent)]"
				/>
			</div>
			<div>
				<label for="invite-role" class="mb-1 block text-sm text-[var(--color-text-secondary)]"
					>{m['settings.members.role']()}</label
				>
				<select
					id="invite-role"
					bind:value={inviteRole}
					class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none"
				>
					<option value="admin">{m['settings.role_admin']()}</option>
					<option value="member">{m['settings.role_member']()}</option>
					<option value="guest">{m['common.role.guest']()}</option>
				</select>
			</div>
			<Dialog.Footer>
				<Button type="button" variant="outline" disabled={adding} onclick={() => (showInvite = false)}
					>{m['settings.cancel']()}</Button
				>
				<Button type="submit" disabled={adding || !inviteEmail.trim()}
					>{adding ? m['sidebar.creating']() : copy.addMember}</Button
				>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
