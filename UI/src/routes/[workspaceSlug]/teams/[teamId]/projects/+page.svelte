<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { listTeamProjects, createProject } from '$lib/api/projects';
	import { listTeams } from '$lib/api/teams';
	import type { Project, ProjectStatus } from '$lib/types/project';
	import type { Team } from '$lib/types/team';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import CreateProjectDialog from '$lib/features/projects/CreateProjectDialog.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { sidebarState } from '$lib/features/layout/sidebar.state.svelte';
	import { appToast } from '$lib/features/toast/toast';
	import { Plus, SquareUser, Box, ChevronRight } from 'lucide-svelte';
	import SidebarToggle from '$lib/components/layout/SidebarToggle.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, setLocale } from '$lib/paraglide/runtime.js';

	const slug = $derived(page.params.workspaceSlug ?? '');
	const teamId = $derived(page.params.teamId ?? '');
	let projects = $state<Project[]>([]);
	let teams = $state<Team[]>([]);
	let loading = $state(true);
	let showCreateProject = $state(false);
	let loadError = $state(false);
	let loadVersion = 0;

	function statusLabel(status: ProjectStatus): string {
		return m[`projects.status.${status}`]();
	}

	async function loadProjects() {
		const s = slug;
		const t = teamId;
		const request = ++loadVersion;
		loading = true;
		loadError = false;
		try {
			const [nextProjects, nextTeams] = await Promise.all([listTeamProjects(s, t), listTeams(s)]);
			if (request !== loadVersion || s !== slug || t !== teamId) return;
			projects = nextProjects;
			teams = nextTeams;
		} catch {
			if (request === loadVersion) loadError = true;
		} finally {
			if (request === loadVersion) loading = false;
		}
	}
	$effect(() => {
		const s = slug;
		const t = teamId;
		if (!s || !t) return;
		void loadProjects();
		return () => {
			loadVersion++;
		};
	});

	async function handleCreate(data: { name: string; description?: string; team_id?: string }) {
		try {
			const project = await createProject(slug, { ...data, team_id: teamId });
			projects = [...projects, project];
			sidebarState.addProject(project);
			appToast.success(m['projects.toast.created']());
		} catch (err: any) {
			appToast.apiError(err, m['projects.toast.failed_create']());
			throw err;
		}
	}

	function progressPercentage(project: Project): number {
		if (!project.progress || project.progress.total === 0) return 0;
		return Math.round((project.progress.completed / project.progress.total) * 100);
	}

	function statusVariant(status: ProjectStatus): 'default' | 'secondary' | 'outline' | 'destructive' {
		switch (status) {
			case 'in_progress':
				return 'default';
			case 'completed':
				return 'secondary';
			case 'cancelled':
				return 'destructive';
			default:
				return 'outline';
		}
	}
</script>

<div class="h-full">
	<div
		class="flex min-h-[49px] items-center justify-between gap-2 border-b border-[var(--app-border)] px-4 py-2 sm:px-6"
	>
		<div class="flex min-w-0 items-center gap-3">
			<SidebarToggle />
			<nav aria-label="Breadcrumb" class="flex min-w-0 flex-wrap items-center gap-1.5 text-sm">
				{#if sidebarState.getTeam(teamId)}
					<a
						href="/{slug}/teams/{teamId}"
						class="flex min-w-0 items-center gap-1.5 break-words [overflow-wrap:anywhere] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
					>
						<SquareUser size={14} class="shrink-0" style="color: {sidebarState.getTeamColor(teamId)}" />
						{sidebarState.getTeam(teamId)?.name}
					</a>
					<ChevronRight size={12} class="shrink-0 text-[var(--color-text-tertiary)]" />
				{/if}
				<span class="flex items-center gap-1.5 font-medium text-[var(--color-text-primary)]">
					<Box size={14} class="shrink-0" />
					{m['projects.title']()}
				</span>
			</nav>
		</div>
		<button
			onclick={() => (showCreateProject = true)}
			class="rounded-md p-1 text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)]"
			title={m['projects.new_project']()}
			aria-label={m['projects.new_project']()}
		>
			<Plus size={16} />
		</button>
	</div>

	{#if loading}
		<p role="status" class="p-8 text-center text-sm text-[var(--color-text-tertiary)]">{m['common.loading']()}</p>
	{:else if loadError}
		<div role="alert" class="p-8 text-center">
			<p class="mb-3 text-sm">
				{getLocale() === 'it' ? 'Impossibile caricare i progetti.' : 'Could not load projects.'}
			</p>
			<button class="rounded-md border border-[var(--app-border)] px-4 py-2 text-sm" onclick={loadProjects}
				>{getLocale() === 'it' ? 'Riprova' : 'Retry'}</button
			>
		</div>
	{:else if projects.length === 0}
		<EmptyState
			title={m['projects.no_team_projects']()}
			description={m['projects.no_team_projects_desc']()}
			action={{ label: m['projects.new_project'](), onclick: () => (showCreateProject = true) }}
		/>
	{:else}
		<div class="divide-y divide-[var(--app-border)]">
			{#each projects as project}
				<div class="flex flex-wrap items-center gap-3 px-4 py-3 sm:px-6 hover:bg-[var(--color-bg-hover)]">
					<a
						href="/{slug}/projects/{project.id}"
						class="flex min-w-0 flex-1 flex-wrap items-center gap-3 rounded-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--app-accent)]"
					>
						<div class="flex-1 min-w-0">
							<div class="flex flex-wrap items-center gap-2">
								<span class="text-sm font-medium text-[var(--color-text-primary)] break-words [overflow-wrap:anywhere]"
									>{project.name}</span
								>
								<Badge variant={statusVariant(project.status)} class="text-[10px]">
									{statusLabel(project.status)}
								</Badge>
							</div>
							{#if project.description}
								<p class="mt-0.5 truncate text-xs text-[var(--color-text-tertiary)]">{project.description}</p>
							{/if}
						</div>
						{#if project.progress && project.progress.total > 0}
							<div class="flex items-center gap-2 shrink-0">
								<div class="relative h-1.5 w-24 overflow-hidden rounded-full bg-[var(--color-bg-tertiary)]">
									<div
										class="absolute left-0 top-0 h-full rounded-full bg-[var(--color-success)]"
										style="width: {project.progress.total > 0
											? (project.progress.completed / project.progress.total) * 100
											: 0}%"
									></div>
								</div>
								<span class="text-xs tabular-nums text-[var(--color-text-tertiary)]">
									{progressPercentage(project)}%
								</span>
							</div>
						{/if}
					</a>
					<a
						href="/{slug}/projects/{project.id}?view=delivery"
						aria-label={getLocale() === 'it' ? `Delivery di ${project.name}` : `Delivery for ${project.name}`}
						class="shrink-0 rounded-md border border-[var(--app-border)] px-3 py-2 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--app-accent)]"
						>Delivery</a
					>
				</div>
			{/each}
		</div>
	{/if}
</div>

<CreateProjectDialog bind:open={showCreateProject} {teams} defaultTeamId={teamId} onsubmit={handleCreate} />
