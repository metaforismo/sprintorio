<script lang="ts">
	import { onMount } from 'svelte';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getProject, updateProject, deleteProject } from '$lib/api/projects';
	import { getWorkspace } from '$lib/api/workspaces';
	import { issuesState } from '$lib/features/issues/issues.state.svelte';
	import { teamStatusesState } from '$lib/features/issues/team-statuses.state.svelte';
	import { listCycles } from '$lib/api/cycles';
	import { listTeams } from '$lib/api/teams';
	import type { Project, ProjectStatus } from '$lib/types/project';
	import type { Cycle } from '$lib/types/cycle';
	import type { Team } from '$lib/types/team';
	import IssueRow from '$lib/features/issues/IssueRow.svelte';
	import IssueListLoadMore from '$lib/features/issues/IssueListLoadMore.svelte';
	import IssueDetail from '$lib/features/issues/IssueDetail.svelte';
	import GanttChart from '$lib/features/projects/GanttChart.svelte';
	import DeliveryPlan from '$lib/features/delivery/DeliveryPlan.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import CycleProgress from '$lib/features/cycles/CycleProgress.svelte';
	import DatePickerPopover from '$lib/components/shared/DatePickerPopover.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Select from '$lib/components/ui/select';
	import * as Popover from '$lib/components/ui/popover';
	import { getGitHubStatus } from '$lib/api/github';
	import {
		deleteDevMachineScopeSetting,
		getDevMachineScopeSetting,
		listDevMachineEnvironments,
		updateDevMachineScopeSetting
	} from '$lib/api/dev-machines';
	import type { GitHubRepo } from '$lib/types/github';
	import type { DevMachineEnvironment } from '$lib/types/dev-machine';
	import { appToast } from '$lib/features/toast/toast';
	import { createKeyboardHandler } from '$lib/utils/keyboard';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, setLocale } from '$lib/paraglide/runtime.js';
	import {
		Trash2,
		MoreHorizontal,
		Circle,
		Play,
		CheckCircle2,
		XCircle,
		Calendar,
		List,
		BarChart3,
		ChevronRight,
		SquareUser,
		Box,
		Settings2,
		Plus
	} from 'lucide-svelte';
	import { sidebarState } from '$lib/features/layout/sidebar.state.svelte';
	import SidebarToggle from '$lib/components/layout/SidebarToggle.svelte';

	const slug = $derived(page.params.workspaceSlug ?? '');
	const projectId = $derived(page.params.projectId ?? '');

	let project = $state<Project | null>(null);
	let teams = $state<Team[]>([]);
	let cycles = $state<Cycle[]>([]);
	let loading = $state(true);
	let statusOpen = $state(false);
	let actionsOpen = $state(false);
	let deleteOpen = $state(false);
	let deleting = $state(false);
	let projectError = $state(false);
	let canManageProject = $state(false);
	let cyclesLoading = $state(false);
	let cyclesError = $state(false);
	let cyclesVersion = 0;
	let viewMode = $state<'list' | 'gantt' | 'delivery'>('list');
	let projectRequestVersion = 0;
	let lastSelectedId = $state<string | null>(null);
	let developmentOpen = $state(false);
	let developmentRepositories = $state<GitHubRepo[]>([]);
	let developmentEnvironments = $state<DevMachineEnvironment[]>([]);
	let developmentRepositoryId = $state('inherit');
	let developmentEnvironmentId = $state('inherit');
	let developmentLoading = $state(true);
	let developmentReady = $state(false);
	let savingDevelopment = $state(false);
	let canManageDevelopment = $state(false);
	let developmentRequestVersion = 0;
	let developmentSaveVersion = 0;
	const developmentEnabled = $derived(authState.user?.dev_machines_enabled !== false);
	const projectTeam = $derived(project?.team_id ? teams.find((t) => t.id === project!.team_id) : null);

	const STATUS_OPTIONS: { value: ProjectStatus; label: string; icon: typeof Circle }[] = [
		{ value: 'planned', label: m['projects.status.planned'](), icon: Circle },
		{ value: 'in_progress', label: m['projects.status.in_progress'](), icon: Play },
		{ value: 'completed', label: m['projects.status.completed'](), icon: CheckCircle2 },
		{ value: 'cancelled', label: m['projects.status.cancelled'](), icon: XCircle }
	];

	function isCurrentDevelopmentScope(s: string, pid: string, version: number) {
		return developmentEnabled && slug === s && projectId === pid && developmentRequestVersion === version;
	}

	async function loadDevelopmentSettings(s: string, pid: string, version: number) {
		try {
			const workspace = await getWorkspace(s);
			if (!isCurrentDevelopmentScope(s, pid, version)) return;
			canManageDevelopment = workspace.current_user_role === 'owner' || workspace.current_user_role === 'admin';
			const [github, developmentSetting, availableEnvironments] = await Promise.all([
				getGitHubStatus(s),
				getDevMachineScopeSetting(s, 'project', pid),
				listDevMachineEnvironments(s)
			]);
			if (!isCurrentDevelopmentScope(s, pid, version)) return;
			developmentRepositories = github.repos ?? [];
			developmentEnvironments = (availableEnvironments ?? []).filter((item) => item.status === 'ready');
			developmentRepositoryId = developmentSetting.github_repo_id ?? 'inherit';
			developmentEnvironmentId = developmentSetting.environment_id ?? 'inherit';
			developmentReady = true;
		} catch (error) {
			if (!isCurrentDevelopmentScope(s, pid, version)) return;
			appToast.apiError(error, m['projects.toast.failed_load_development']());
		} finally {
			if (isCurrentDevelopmentScope(s, pid, version)) developmentLoading = false;
		}
	}

	async function loadProject(s: string, pid: string, request: number) {
		loading = true;
		projectError = false;
		project = null;
		try {
			const [nextProject, workspace] = await Promise.all([getProject(s, pid), getWorkspace(s)]);
			if (request !== projectRequestVersion) return;
			project = nextProject;
			canManageProject = ['owner', 'admin', 'member'].includes(workspace.current_user_role);
			await issuesState.load(s, viewMode === 'gantt' ? { project: pid, per_page: '200' } : { project: pid });
			if (request !== projectRequestVersion) return;
			const firstTeamId = issuesState.issues[0]?.team_id;
			if (firstTeamId) {
				teamStatusesState.load(s, firstTeamId);
			}
			const nextTeams = await listTeams(s);
			if (request !== projectRequestVersion) return;
			teams = nextTeams;
		} catch {
			if (request !== projectRequestVersion) return;
			projectError = true;
		} finally {
			if (request === projectRequestVersion) loading = false;
		}
	}

	$effect(() => {
		const request = ++projectRequestVersion;
		void loadProject(slug, projectId, request);
		return () => {
			if (request === projectRequestVersion) projectRequestVersion++;
		};
	});

	$effect(() => {
		const s = slug;
		const pid = projectId;
		const version = ++developmentRequestVersion;
		developmentSaveVersion++;
		developmentRepositories = [];
		developmentEnvironments = [];
		developmentRepositoryId = 'inherit';
		developmentEnvironmentId = 'inherit';
		developmentLoading = true;
		developmentReady = false;
		savingDevelopment = false;
		canManageDevelopment = false;
		if (!developmentEnabled) {
			developmentOpen = false;
			return;
		}
		if (!s || !pid || !developmentOpen) return;
		void loadDevelopmentSettings(s, pid, version);
		return () => {
			if (developmentRequestVersion === version) developmentRequestVersion++;
			developmentSaveVersion++;
		};
	});

	async function loadTimeline() {
		const request = ++cyclesVersion;
		const s = slug;
		const pid = projectId;
		cyclesLoading = true;
		cyclesError = false;
		try {
			await issuesState.load(s, { project: pid, per_page: '200' });
			if (request !== cyclesVersion || s !== slug || pid !== projectId) return;
			const teamIds = [
				...new Set(
					[project?.team_id, ...issuesState.issues.map((issue) => issue.team_id)].filter((id): id is string => !!id)
				)
			];
			const next = await Promise.all(teamIds.map((id) => listCycles(s, id)));
			if (request === cyclesVersion && s === slug && pid === projectId) cycles = next.flat();
		} catch {
			if (request === cyclesVersion) cyclesError = true;
		} finally {
			if (request === cyclesVersion) cyclesLoading = false;
		}
	}
	$effect(() => {
		if (viewMode !== 'gantt' || !project) return;
		void loadTimeline();
		return () => {
			cyclesVersion++;
		};
	});

	async function handleStatusChange(status: ProjectStatus) {
		if (!project || !canManageProject) return;
		const s = slug,
			pid = project.id,
			request = projectRequestVersion;
		try {
			const updated = await updateProject(s, pid, { status });
			if (s !== slug || pid !== projectId || request !== projectRequestVersion) return;
			project = updated;
			statusOpen = false;
			appToast.success(m['projects.toast.status_updated']());
		} catch (err: any) {
			if (s === slug && pid === projectId && request === projectRequestVersion)
				appToast.apiError(err, m['projects.toast.failed_update_status']());
		}
	}

	async function handleDateChange(field: 'start_date' | 'target_date', value: string | null) {
		if (!project || !canManageProject) return;
		const s = slug,
			pid = project.id,
			request = projectRequestVersion;
		try {
			const updated = await updateProject(s, pid, { [field]: value });
			if (s !== slug || pid !== projectId || request !== projectRequestVersion) return;
			project = updated;
			appToast.success(m['projects.toast.date_updated']());
		} catch (err: any) {
			if (s === slug && pid === projectId && request === projectRequestVersion)
				appToast.apiError(err, m['projects.toast.failed_update_date']());
		}
	}

	async function handleDelete() {
		if (!project || deleting || !canManageProject) return;
		deleting = true;
		try {
			await deleteProject(slug, project.id);
			appToast.success(m['projects.toast.deleted']());
			goto(`/${slug}/projects`);
		} catch (err: any) {
			appToast.apiError(err, m['projects.toast.failed_delete']());
		} finally {
			deleting = false;
		}
	}

	async function saveDevelopmentSettings() {
		if (!developmentEnabled || !canManageDevelopment || developmentLoading || !developmentReady || savingDevelopment)
			return;
		const s = slug;
		const pid = projectId;
		const requestVersion = developmentRequestVersion;
		const saveVersion = ++developmentSaveVersion;
		const repositoryId = developmentRepositoryId;
		const environmentId = developmentEnvironmentId;
		const repository = developmentRepositories.find((item) => item.id === repositoryId);
		savingDevelopment = true;
		try {
			if (repositoryId === 'inherit' && environmentId === 'inherit') {
				await deleteDevMachineScopeSetting(s, 'project', pid);
			} else {
				await updateDevMachineScopeSetting(s, {
					scope_type: 'project',
					scope_id: pid,
					github_repo_id: repository?.id,
					base_branch: repository?.default_branch,
					environment_id: environmentId === 'inherit' ? undefined : environmentId
				});
			}
			if (!isCurrentDevelopmentScope(s, pid, requestVersion) || developmentSaveVersion !== saveVersion) return;
			developmentOpen = false;
			appToast.success(m['projects.toast.development_saved']());
		} catch (error) {
			if (!isCurrentDevelopmentScope(s, pid, requestVersion) || developmentSaveVersion !== saveVersion) return;
			appToast.apiError(error, m['projects.toast.failed_save_development']());
		} finally {
			if (isCurrentDevelopmentScope(s, pid, requestVersion) && developmentSaveVersion === saveVersion)
				savingDevelopment = false;
		}
	}

	const keyHandler = createKeyboardHandler([
		{ key: 'a', ctrl: true, handler: () => issuesState.selectAll() },
		{ key: 'Escape', handler: () => issuesState.clearSelection() }
	]);

	onMount(() => {
		document.addEventListener('keydown', keyHandler);
		return () => document.removeEventListener('keydown', keyHandler);
	});

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

<div class="flex h-full flex-col">
	{#if !loading && project && !projectError}
		<!-- Header -->
		<div
			class="flex min-h-[49px] flex-wrap items-center justify-between gap-2 border-b border-[var(--app-border)] px-4 py-2 sm:px-6"
		>
			<div class="flex min-w-0 flex-wrap items-center gap-3">
				<SidebarToggle />
				<nav aria-label="Breadcrumb" class="flex min-w-0 flex-wrap items-center gap-1.5 text-sm">
					{#if projectTeam}
						<a
							href="/{slug}/teams/{projectTeam.id}"
							class="flex items-center gap-1.5 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
						>
							<SquareUser size={14} class="shrink-0" style="color: {sidebarState.getTeamColor(projectTeam.id)}" />
							{projectTeam.name}
						</a>
						<ChevronRight size={12} class="shrink-0 text-[var(--color-text-tertiary)]" />
						<a
							href="/{slug}/teams/{projectTeam.id}/projects"
							class="flex items-center gap-1.5 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
						>
							<Box size={14} class="shrink-0" />
							{m['projects.title']()}
						</a>
						<ChevronRight size={12} class="shrink-0 text-[var(--color-text-tertiary)]" />
					{/if}
					<span class="font-medium break-words text-[var(--color-text-primary)]">{project.name}</span>
				</nav>
				<Popover.Root bind:open={statusOpen}>
					<Popover.Trigger disabled={!canManageProject}>
						<Badge variant={statusVariant(project.status)} class="cursor-pointer text-[10px]">
							{m[`projects.status.${project.status}`]()}
						</Badge>
					</Popover.Trigger>
					<Popover.Content class="w-40 p-1" align="start">
						{#each STATUS_OPTIONS as option}
							{@const StatusIcon = option.icon}
							<button
								onclick={() => handleStatusChange(option.value)}
								class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] {project.status ===
								option.value
									? 'bg-[var(--color-bg-hover)]'
									: ''}"
							>
								<StatusIcon size={14} />
								{option.label}
							</button>
						{/each}
					</Popover.Content>
				</Popover.Root>
			</div>
			<div class="flex max-w-full flex-wrap items-center gap-2">
				{#if developmentEnabled}
					<Button
						variant="ghost"
						size="icon-sm"
						aria-label={m['projects.development.settings']()}
						onclick={() => (developmentOpen = true)}
						title={m['projects.development.settings']()}><Settings2 size={15} /></Button
					>
				{/if}
				<Button
					disabled={!canManageProject}
					variant="outline"
					size="sm"
					onclick={() => window.dispatchEvent(new CustomEvent('sprintorio:create-issue'))}
					><Plus size={14} />{m['sidebar.create_issue']()}</Button
				>
				<!-- View switcher -->
				<div class="flex rounded-md border border-[var(--app-border)]">
					<button
						onclick={() => (viewMode = 'list')}
						aria-label="Issue list"
						aria-pressed={viewMode === 'list'}
						class="flex items-center rounded-l-md px-2 py-2 {viewMode === 'list'
							? 'bg-[var(--color-bg-hover)] text-[var(--color-text-primary)]'
							: 'text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)]'}"
						title={m['projects.list_view']()}
					>
						<List size={14} /><span class="ml-1.5 text-xs">{getLocale() === 'it' ? 'Attività' : 'Issues'}</span>
					</button>
					<button
						onclick={() => (viewMode = 'gantt')}
						aria-label="Gantt chart"
						aria-pressed={viewMode === 'gantt'}
						class="flex items-center px-2 py-2 {viewMode === 'gantt'
							? 'bg-[var(--color-bg-hover)] text-[var(--color-text-primary)]'
							: 'text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)]'}"
						title={m['projects.gantt_view']()}
					>
						<BarChart3 size={14} /><span class="ml-1.5 text-xs">Timeline</span>
					</button>
					<button
						onclick={() => (viewMode = 'delivery')}
						aria-pressed={viewMode === 'delivery'}
						class="rounded-r-md px-3 py-2 text-xs {viewMode === 'delivery'
							? 'bg-[var(--color-bg-hover)] text-[var(--color-text-primary)]'
							: 'text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)]'}">Delivery</button
					>
				</div>

				<Popover.Root bind:open={actionsOpen}>
					<Popover.Trigger>
						{#snippet child({ props })}
							<Button
								{...props}
								disabled={!canManageProject}
								variant="ghost"
								size="icon-sm"
								aria-label={getLocale() === 'it' ? 'Azioni progetto' : 'Project actions'}
							>
								<MoreHorizontal size={14} />
							</Button>
						{/snippet}
					</Popover.Trigger>
					<Popover.Content class="w-40 p-1" align="end">
						<button
							onclick={() => {
								actionsOpen = false;
								deleteOpen = true;
							}}
							class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-[var(--color-error)] hover:bg-[var(--color-bg-hover)]"
						>
							<Trash2 size={14} />
							{m['projects.delete']()}
						</button>
					</Popover.Content>
				</Popover.Root>
			</div>
		</div>

		<!-- Project info -->
		<div class="border-b border-[var(--app-border)] px-6 py-4">
			<div class="flex flex-wrap items-center gap-4 text-xs text-[var(--color-text-tertiary)]">
				<div class="flex items-center gap-1.5">
					<Calendar size={12} />
					<span>{m['projects.start_date']()}</span>
					<DatePickerPopover
						value={project.start_date}
						onchange={(d) => handleDateChange('start_date', d)}
						placeholder={m['projects.set_start']()}
					/>
				</div>
				<div class="flex items-center gap-1.5">
					<span>{m['projects.target_date']()}</span>
					<DatePickerPopover
						value={project.target_date}
						onchange={(d) => handleDateChange('target_date', d)}
						placeholder={m['projects.set_target']()}
					/>
				</div>
				{#if project.progress}
					<span>{m['projects.progress']({ completed: project.progress.completed, total: project.progress.total })}</span
					>
				{/if}
			</div>
			{#if project.description}
				<p class="mt-2 text-sm text-[var(--color-text-secondary)]">{project.description}</p>
			{/if}
			{#if project.progress && project.progress.total > 0}
				<div class="mt-3 w-64">
					<CycleProgress progress={project.progress} />
				</div>
			{/if}
		</div>

		<!-- Keep the plan mounted across view switches to retain unsaved edits. -->
		<div hidden={viewMode !== 'delivery'} class="min-h-0 flex-1 overflow-y-auto">
			<DeliveryPlan {slug} {projectId} />
		</div>
		<!-- Content -->
		{#if viewMode === 'list'}
			<div class="flex-1 overflow-y-auto">
				{#if !issuesState.loading && issuesState.issues.length === 0}
					<EmptyState
						title={m['projects.no_issues']()}
						description={m['projects.no_issues_desc']()}
						action={canManageProject
							? {
									label: m['sidebar.create_issue'](),
									onclick: () => window.dispatchEvent(new CustomEvent('sprintorio:create-issue'))
								}
							: undefined}
					/>
				{:else}
					{#each issuesState.issues as issue (issue.id)}
						<IssueRow
							{issue}
							{slug}
							{lastSelectedId}
							onlastselected={(id) => (lastSelectedId = id)}
							onclick={(i) => {
								lastSelectedId = i.id;
								issuesState.select(i);
							}}
						/>
					{/each}
				{/if}
				<IssueListLoadMore />
			</div>
		{:else if viewMode === 'gantt'}
			<div class="flex-1 min-h-0 px-4 py-3">
				{#if cyclesLoading}
					<p role="status" class="p-4 text-sm">{m['common.loading']()}</p>
				{:else if cyclesError}
					<div role="alert" class="p-4 text-sm">
						<p>{getLocale() === 'it' ? 'Impossibile caricare la timeline.' : 'Could not load the timeline.'}</p>
						<Button class="mt-2" variant="outline" onclick={loadTimeline}
							>{getLocale() === 'it' ? 'Riprova' : 'Retry'}</Button
						>
					</div>
				{:else if !issuesState.loading}
					{#if issuesState.hasMore}<p role="status" class="mb-3 text-xs text-[var(--color-text-tertiary)]">
							{getLocale() === 'it'
								? `Timeline parziale: ${issuesState.issues.length} di ${issuesState.totalCount} attività. Usa l'elenco per caricare le altre.`
								: `Partial timeline: ${issuesState.issues.length} of ${issuesState.totalCount} issues. Use the issue list to load more.`}
						</p>{/if}
					<GanttChart
						issues={issuesState.issues}
						{cycles}
						onissueclick={(i) => goto(`/${slug}/issue/${i.identifier}`)}
					/>
				{/if}
			</div>
		{/if}
	{:else if loading}<p role="status" class="p-8 text-center text-sm">{m['common.loading']()}</p>
	{:else if projectError}<div role="alert" class="p-8 text-center">
			<p class="mb-3 text-sm">
				{getLocale() === 'it' ? 'Impossibile caricare il progetto.' : 'Could not load this project.'}
			</p>
			<Button variant="outline" onclick={() => loadProject(slug, projectId, ++projectRequestVersion)}
				>{getLocale() === 'it' ? 'Riprova' : 'Retry'}</Button
			>
		</div>
	{/if}
</div>

<Dialog.Root bind:open={deleteOpen}>
	<Dialog.Content
		><Dialog.Header
			><Dialog.Title>{getLocale() === 'it' ? 'Eliminare il progetto?' : 'Delete project?'}</Dialog.Title
			><Dialog.Description
				>{project?.name}. {getLocale() === 'it'
					? 'Il brief, le milestone e i test verranno eliminati. Le attività rimarranno nel workspace.'
					: 'The brief, milestones and tests will be deleted. Issues remain in the workspace.'}</Dialog.Description
			></Dialog.Header
		><Dialog.Footer
			><Button variant="outline" disabled={deleting} onclick={() => (deleteOpen = false)}>{m['common.cancel']()}</Button
			><Button variant="destructive" disabled={deleting} onclick={handleDelete}
				>{deleting ? m['common.saving']() : m['projects.delete']()}</Button
			></Dialog.Footer
		></Dialog.Content
	>
</Dialog.Root>

{#if developmentEnabled}
	<Dialog.Root bind:open={developmentOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header
				><Dialog.Title>{m['projects.development.title']()}</Dialog.Title><Dialog.Description
					>{m['projects.development.description']()}</Dialog.Description
				></Dialog.Header
			>
			{#if !developmentLoading && !canManageDevelopment}<p
					class="rounded-md border border-[var(--app-border)] p-3 text-xs text-[var(--color-text-tertiary)]"
				>
					{m['projects.development.permission_note']()}
				</p>{/if}
			<div class="space-y-4">
				<div class="space-y-1">
					<Label>{m['projects.development.repository']()}</Label><Select.Root
						type="single"
						value={developmentRepositoryId}
						disabled={developmentLoading || !developmentReady || !canManageDevelopment}
						onValueChange={(value) => value && (developmentRepositoryId = value)}
						><Select.Trigger class="w-full"
							>{developmentLoading
								? m['common.loading']()
								: (developmentRepositories.find((item) => item.id === developmentRepositoryId)?.full_name ??
									m['projects.development.team_or_workspace_default']())}</Select.Trigger
						><Select.Content
							><Select.Item value="inherit" label={m['projects.development.inherited_default']()}
								>{m['projects.development.inherited_default']()}</Select.Item
							>{#each developmentRepositories as repository}<Select.Item
									value={repository.id}
									label={repository.full_name}>{repository.full_name}</Select.Item
								>{/each}</Select.Content
						></Select.Root
					>
				</div>
				<div class="space-y-1">
					<Label>{m['projects.development.environment']()}</Label><Select.Root
						type="single"
						value={developmentEnvironmentId}
						disabled={developmentLoading || !developmentReady || !canManageDevelopment}
						onValueChange={(value) => value && (developmentEnvironmentId = value)}
						><Select.Trigger class="w-full"
							>{developmentLoading
								? m['common.loading']()
								: (developmentEnvironments.find((item) => item.id === developmentEnvironmentId)?.name ??
									m['projects.development.team_or_workspace_default']())}</Select.Trigger
						><Select.Content
							><Select.Item value="inherit" label={m['projects.development.inherited_default']()}
								>{m['projects.development.inherited_default']()}</Select.Item
							>{#each developmentEnvironments as environment}<Select.Item
									value={environment.id}
									label={environment.name}>{environment.name}</Select.Item
								>{/each}</Select.Content
						></Select.Root
					>
				</div>
			</div>
			<Dialog.Footer
				><Button variant="outline" onclick={() => (developmentOpen = false)}>{m['common.cancel']()}</Button><Button
					onclick={saveDevelopmentSettings}
					disabled={developmentLoading || !developmentReady || savingDevelopment || !canManageDevelopment}
					>{savingDevelopment ? m['common.saving']() : m['common.save']()}</Button
				></Dialog.Footer
			>
		</Dialog.Content>
	</Dialog.Root>
{/if}

{#if issuesState.selectedIssue}
	<IssueDetail issue={issuesState.selectedIssue} {slug} onclose={() => issuesState.select(null)} />
{/if}
