<script lang="ts">
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { deliveryText } from './delivery-copy';
	import { deliveryContext, deliveryReadiness } from './delivery-context';
	import { beforeNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import InfoPopover from '$lib/components/shared/InfoPopover.svelte';
	import TestCaseEditor from './TestCaseEditor.svelte';
	import ChoiceSelect from './ChoiceSelect.svelte';
	import TestResult from './TestResult.svelte';
	import * as Menu from '$lib/components/ui/dropdown-menu';
	import { appToast } from '$lib/features/toast/toast';
	import { getDeliveryPlan, saveDeliveryPlan } from '$lib/api/delivery-plan';
	import { getWorkspace } from '$lib/api/workspaces';
	import { hasPermission } from '$lib/security/permissions';
	import type { Role } from '$lib/security/roles';
	import { emptyDeliveryPlan, type DeliveryPlan } from '$lib/types/delivery-plan';
	import { Button } from '$lib/components/ui/button';
	import {
		Plus,
		Trash2,
		ClipboardCheck,
		Flag,
		Package,
		Copy,
		MoreHorizontal,
		Check,
		Search
	} from 'lucide-svelte';

	let { slug, projectId }: { slug: string; projectId: string } = $props();
	let plan = $state<DeliveryPlan>(emptyDeliveryPlan());
	let saved = $state('');
	let version = $state(0);
	let loading = $state(true);
	let saving = $state(false);
	let canEdit = $state(false);
	let error = $state('');
	let conflict = $state(false);
	let copying = $state(false);
	let generation = 0;
	let editingTest = $state<DeliveryPlan['test_cases'][number] | null>(null);
	let creatingTest = $state(false);
	let copied = $state(false);
	let testingSection = $state<HTMLElement | null>(null);
	const feedbackId = 'delivery-context-copy';
	let testSearch = $state('');
	let testFilter = $state('all');
	const visibleTests = $derived(
		plan.test_cases.filter(
			(test) =>
				(!testSearch.trim() || `${test.title} ${test.steps}`.toLowerCase().includes(testSearch.trim().toLowerCase())) &&
				(testFilter === 'all' || test.status === testFilter)
		)
	);
	const t = (text: string) => deliveryText(text, getLocale());
	const testStarters = {
		blank: null,
		acceptance: {
			title: t('Acceptance check'),
			steps: t(
				'Use the product as the intended user and complete the main workflow. Replace these steps with your release criteria.'
			),
			expected: t('The user completes the workflow and the success metric is met.')
		},
		regression: {
			title: t('Regression check'),
			steps: t('Repeat a previously working workflow affected by this release. Record the environment and test data.'),
			expected: t('Existing behavior remains correct and saved data is preserved.')
		},
		accessibility: {
			title: t('Accessibility check'),
			steps: t(
				'Complete the main workflow using only the keyboard. Check visible focus, field labels and announcements with a screen reader.'
			),
			expected: t(
				'Every action is reachable, focus stays visible, and controls and errors have clear accessible names.'
			)
		}
	};

	const dirty = $derived(saved !== '' && JSON.stringify(plan) !== saved);
	const passed = $derived(plan.test_cases.filter((t) => t.status === 'passed').length);
	const failed = $derived(plan.test_cases.filter((t) => t.status === 'failed').length);
	const blocked = $derived(plan.test_cases.filter((t) => t.status === 'blocked').length);
	const pending = $derived(plan.test_cases.filter((t) => t.status === 'not_run').length);
	const completed = $derived(plan.milestones.filter((m) => m.status === 'done').length);
	const readiness = $derived(t(deliveryReadiness(plan)));
	const fieldClass =
		'w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg-primary)] px-3 py-2 text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] disabled:opacity-70';

	async function load(s = slug, id = projectId, request = ++generation) {
		loading = true;
		error = '';
		try {
			const [result, workspace] = await Promise.all([getDeliveryPlan(s, id), getWorkspace(s)]);
			if (request !== generation || s !== slug || id !== projectId) return;
			plan = result.plan;
			version = result.version;
			saved = JSON.stringify(result.plan);
			canEdit = hasPermission(workspace.current_user_role as Role, 'project:manage');
			conflict = false;
		} catch {
			if (request === generation) error = t('Could not load the delivery plan. Try again.');
		} finally {
			if (request === generation) loading = false;
		}
	}
	$effect(() => {
		const s = slug;
		const id = projectId;
		const request = ++generation;
		editingTest = null;
		copied = false;
		plan = emptyDeliveryPlan();
		saved = '';
		canEdit = false;
		copying = false;
		saving = false;
		conflict = false;
		void load(s, id, request);
		return () => {
			if (generation === request) generation++;
		};
	});
	beforeNavigate(({ cancel, to, willUnload }) => {
		if (!dirty || (!willUnload && to?.url.pathname === page.url.pathname)) return;
		if (!window.confirm(t('Leave this project and discard unsaved delivery changes?'))) cancel();
	});
	function preventUnload(event: BeforeUnloadEvent) {
		if (dirty) {
			event.preventDefault();
			event.returnValue = '';
		}
	}
	function cancelChanges() {
		plan = JSON.parse(saved);
		error = conflict ? t('Someone saved a newer plan. Reload the latest version before editing again.') : '';
	}
	async function copyContext() {
		if (copying || saving || dirty || saved === '' || conflict) return;
		const request = generation;
		copying = true;
		copied = false;
		appToast.dismiss(feedbackId);
		try {
			await navigator.clipboard.writeText(JSON.stringify(deliveryContext(slug, projectId, version, JSON.parse(saved))));
			if (request === generation) {
				copied = true;
				appToast.success(t('Saved context copied.'), { id: feedbackId });
			}
		} catch {
			if (request === generation) appToast.error(t('Could not copy. Try again.'), { id: feedbackId });
		} finally {
			if (request === generation) copying = false;
		}
	}
	function addMilestone() {
		plan.milestones.push({ id: crypto.randomUUID(), title: '', due_date: '', status: 'planned' });
	}
	function editTest(testCase: DeliveryPlan['test_cases'][number]) {
		creatingTest = false;
		editingTest = JSON.parse(JSON.stringify(testCase));
	}
	function addTest(template: 'blank' | 'acceptance' | 'regression' | 'accessibility' = 'blank') {
		creatingTest = true;
		const starter = testStarters[template];
		editingTest = {
			id: crypto.randomUUID(),
			title: starter?.title ?? '',
			steps: starter?.steps ?? '',
			expected_result: starter?.expected ?? '',
			status: 'not_run',
			evidence: ''
		};
	}
	function applyTest(testCase: DeliveryPlan['test_cases'][number]) {
		if (creatingTest) plan.test_cases = [...plan.test_cases, testCase];
		else plan.test_cases = plan.test_cases.map((item) => (item.id === testCase.id ? testCase : item));
		editingTest = null;
		testSearch = '';
		testFilter = 'all';
	}
	function duplicateTest(testCase: DeliveryPlan['test_cases'][number]) {
		creatingTest = true;
		editingTest = {
			...testCase,
			id: crypto.randomUUID(),
			title: `${testCase.title} (${t('Copy')})`,
			status: 'not_run',
			evidence: ''
		};
	}
	$effect(() => {
		if (!loading && page.url.searchParams.get('section') === 'testing' && testingSection)
			testingSection.scrollIntoView({ block: 'start' });
	});
	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (!canEdit || saving || !dirty || conflict) return;
		const s = slug;
		const id = projectId;
		const request = generation;
		const snapshot = JSON.parse(JSON.stringify(plan)) as DeliveryPlan;
		if (snapshot.milestones.some((m) => !m.title.trim()) || snapshot.test_cases.some((t) => !t.title.trim())) {
			error = t('Give every milestone and test case a title before saving.');
			return;
		}
		if (snapshot.test_cases.some((t) => (t.status === 'passed' || t.status === 'failed') && !t.evidence.trim())) {
			error = t('Passed and failed tests need recorded evidence before saving.');
			return;
		}
		saving = true;
		error = '';
		try {
			const result = await saveDeliveryPlan(s, id, snapshot, version);
			if (request !== generation || s !== slug || id !== projectId) return;
			plan = result.plan;
			saved = JSON.stringify(result.plan);
			version = result.version;
			appToast.success(t('Delivery plan saved.'));
			copied = false;
		} catch (err) {
			if (request !== generation) return;
			const code = (err as { error?: { code?: string } })?.error?.code;
			conflict = !!code && /conflict/i.test(code);
			error = conflict
				? t('Someone saved a newer plan. Your edits are still here. Copy anything you want to keep before reloading.')
				: t('Could not save the delivery plan. Your edits are still here. Try again.');
		} finally {
			if (request === generation) saving = false;
		}
	}
</script>

<svelte:window onbeforeunload={preventUnload} />
<div class="mx-auto w-full max-w-5xl px-4 py-6 sm:px-8">
	{#if loading}
		<p role="status" class="py-12 text-center text-sm text-[var(--color-text-tertiary)]">
			{t('Loading delivery plan…')}
		</p>
	{:else if saved === ''}
		<div role="alert" class="rounded-lg border border-[var(--app-border)] p-6">
			<p class="mb-4">{error}</p>
			<Button variant="outline" onclick={() => load()}>{t('Retry loading')}</Button>
		</div>
	{:else}
		<form onsubmit={save} class="space-y-8">
			<div class="flex flex-wrap items-start justify-between gap-4">
				<div>
					<h1 class="text-xl font-semibold">{t('Product delivery')}</h1>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<Button
						type="button"
						variant="outline"
						class="min-h-11"
						disabled={dirty || saving || copying || conflict}
						onclick={copyContext}
						><span class="flex w-4 justify-center"
							>{#if copied}<Check size={14} />{:else}<Copy size={14} />{/if}</span
						>{t('Copy context')}</Button
					>
					<span class="w-28 text-xs text-[var(--color-text-tertiary)]" aria-live="polite"
						>{dirty ? t('Unsaved changes') : ''}</span
					>
					{#if canEdit}<Button type="button" variant="outline" disabled={!dirty || saving} onclick={cancelChanges}
							>{t('Cancel changes')}</Button
						><Button type="submit" disabled={!dirty || saving || conflict}
							>{saving ? t('Saving…') : t('Save plan')}</Button
						>{/if}
				</div>
			</div>
			{#if !canEdit}<p class="text-sm text-[var(--color-text-tertiary)]">
					{t('View only. Workspace owners, admins, and members can edit this plan.')}
				</p>{/if}
			{#if error}<div role="alert" class="rounded-lg border border-[var(--color-error)] p-4 text-sm">
					<p>{error}</p>
					{#if conflict}<Button type="button" variant="outline" class="mt-3" onclick={() => load()}
							>{t('Reload latest and discard my edits')}</Button
						>{/if}
				</div>{/if}

			<div
				class="flex flex-wrap items-center justify-between gap-4 border-y border-[var(--app-border)] py-4"
				data-testid="delivery-readiness"
			>
				<div class="flex items-center gap-2 text-sm font-medium">
					<ClipboardCheck size={16} /><span>{readiness}</span>
					<InfoPopover label={t('How readiness works')} title={t('Ready for review')}
						><p>
							{t(
								'Complete the brief and milestones. Every test needs steps, expected results and recorded evidence. Release approval is separate.'
							)}
						</p></InfoPopover
					>
				</div>
				<div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-[var(--color-text-tertiary)]">
					<span>{completed}/{plan.milestones.length} {t('Milestones done')}</span>
					{#each [{ value: 'passed', count: passed, label: t('Passed') }, { value: 'failed', count: failed, label: t('Failed') }, { value: 'blocked', count: blocked, label: t('Blocked') }, { value: 'not_run', count: pending, label: t('Not run') }] as result}
						{#if result.count}<button
								type="button"
								class="min-h-11 hover:text-[var(--color-text-primary)]"
								onclick={() => {
									testFilter = result.value;
									testingSection?.scrollIntoView({ block: 'start' });
								}}
								aria-label={`${result.count} ${result.label}`}>{result.count} {result.label}</button
							>{/if}
					{/each}
				</div>
			</div>
			<div class="space-y-8">
				<section class="border-b border-[var(--app-border)] pb-6">
					<h2 class="mb-5 flex items-center gap-2 font-medium"><Package size={18} />{t('Product brief')}</h2>
					<div class="grid gap-4 sm:grid-cols-2">
						<label class="space-y-1.5 text-sm"
							><span>{t('Product name')}</span><input
								class={fieldClass}
								disabled={!canEdit || saving}
								maxlength="200"
								bind:value={plan.product_name}
								placeholder={t('What are you delivering?')}
							/></label
						>
						<label class="space-y-1.5 text-sm"
							><span>{t('Target release')}</span><input
								type="date"
								class={fieldClass}
								disabled={!canEdit || saving}
								bind:value={plan.target_release}
							/></label
						>
						<label class="space-y-1.5 text-sm sm:col-span-2"
							><span>{t('Objective')}</span><textarea
								rows="3"
								class={fieldClass}
								disabled={!canEdit || saving}
								maxlength="5000"
								bind:value={plan.objective}
								placeholder={t('The problem this release should solve')}
							></textarea></label
						>
						<label class="space-y-1.5 text-sm sm:col-span-2"
							><span>{t('Success metric')}</span><textarea
								rows="2"
								class={fieldClass}
								disabled={!canEdit || saving}
								maxlength="2000"
								bind:value={plan.success_metric}
								placeholder={t('How will you know it worked?')}
							></textarea></label
						>
					</div>
				</section>
				<section class="border-b border-[var(--app-border)] pb-6">
					<div class="mb-4 flex items-center justify-between gap-3">
						<h2 class="flex flex-wrap items-center gap-2 font-medium"><Flag size={18} />{t('Milestones')}</h2>
						{#if canEdit}<Button
								type="button"
								variant="outline"
								size="sm"
								disabled={saving || plan.milestones.length >= 200}
								onclick={addMilestone}><Plus size={14} />{t('Add milestone')}</Button
							>{/if}
					</div>
					{#if !plan.milestones.length}<p class="py-4 text-sm text-[var(--color-text-tertiary)]">
							{t('No milestones yet.')}
						</p>{/if}
					<div class="space-y-3">
						{#each plan.milestones as milestone, i (milestone.id)}
							<div
								class="grid items-end gap-3 border-b border-[var(--app-border)] py-3 sm:grid-cols-2 lg:grid-cols-[1fr_150px_150px_auto]"
							>
								<label class="space-y-1 text-sm"
									><span>Milestone {i + 1}</span><input
										class={fieldClass}
										disabled={!canEdit || saving}
										required
										maxlength="300"
										bind:value={milestone.title}
									/></label
								>
								<label class="space-y-1 text-sm"
									><span>{t('Due date')}</span><input
										type="date"
										class={fieldClass}
										disabled={!canEdit || saving}
										bind:value={milestone.due_date}
									/></label
								>
								<label class="space-y-1 text-sm"
									><span>{t('Milestone status')}</span><ChoiceSelect
										label={t('Milestone status')}
										value={milestone.status}
										disabled={!canEdit || saving}
										options={[
											{ value: 'planned', label: t('Planned') },
											{ value: 'in_progress', label: t('In progress') },
											{ value: 'done', label: t('Done') }
										]}
										onchange={(value) => (milestone.status = value as typeof milestone.status)}
									/></label
								>
								{#if canEdit}<Button
										type="button"
										variant="ghost"
										disabled={saving}
										aria-label={`${getLocale() === 'it' ? 'Rimuovi milestone' : 'Remove milestone'} ${i + 1}`}
										onclick={() => (plan.milestones = plan.milestones.filter((m) => m.id !== milestone.id))}
										><Trash2 size={16} /></Button
									>{/if}
							</div>
						{/each}
					</div>
				</section>
				<section id="testing" bind:this={testingSection} class="scroll-mt-36" aria-label={t('Manual tests')}>
					<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
						<h2 class="flex items-center gap-2 font-medium">
							{t('Manual tests')}
							<span class="text-xs text-[var(--color-text-tertiary)]">{plan.test_cases.length}</span>
						</h2>
						{#if canEdit}<div class="flex items-center gap-1">
								<Button
									variant="outline"
									class="min-h-11"
									disabled={saving || plan.test_cases.length >= 200}
									onclick={() => addTest()}><Plus size={14} />{t('Add test case')}</Button
								>
								<Menu.Root
									><Menu.Trigger
										aria-label={t('Test templates')}
										class="flex min-h-11 min-w-11 items-center justify-center rounded-md hover:bg-[var(--color-bg-hover)]"
										disabled={saving || plan.test_cases.length >= 200}><MoreHorizontal size={16} /></Menu.Trigger
									><Menu.Content align="end">
										{#each ['acceptance', 'regression', 'accessibility'] as template}<Menu.Item
												onclick={() => addTest(template as 'acceptance' | 'regression' | 'accessibility')}
												>{t(
													template === 'acceptance'
														? 'Acceptance'
														: template === 'regression'
															? 'Regression'
															: 'Accessibility'
												)}</Menu.Item
											>{/each}
									</Menu.Content></Menu.Root
								>
							</div>{/if}
					</div>
					{#if plan.test_cases.length > 3 || testSearch.trim() || testFilter !== 'all'}<div
							class="mb-3 grid gap-2 sm:grid-cols-[1fr_180px]"
						>
							<div class="relative">
								<Search
									size={14}
									class="pointer-events-none absolute left-3 top-3.5 text-[var(--color-text-tertiary)]"
								/><input
									aria-label={t('Search tests')}
									placeholder={t('Search tests')}
									class={`${fieldClass} min-h-11 pl-9`}
									bind:value={testSearch}
								/>
							</div>
							<ChoiceSelect
								label={t('Filter by result')}
								value={testFilter}
								options={[
									{ value: 'all', label: t('All results') },
									{ value: 'not_run', label: t('Not run') },
									{ value: 'passed', label: t('Passed') },
									{ value: 'failed', label: t('Failed') },
									{ value: 'blocked', label: t('Blocked') }
								]}
								onchange={(value) => (testFilter = value)}
							/>
						</div>{/if}
					{#if !plan.test_cases.length}<p class="py-8 text-sm text-[var(--color-text-tertiary)]">
							{t('Add a test case to assess readiness.')}
						</p>
					{:else if !visibleTests.length}<p class="py-8 text-sm text-[var(--color-text-tertiary)]">
							{t('No matching tests.')}
						</p>{/if}
					<div class="divide-y divide-[var(--app-border)] border-y border-[var(--app-border)]">
						{#each visibleTests as testCase (testCase.id)}<div
								class="group flex min-h-14 items-center gap-2"
								data-testid="test-row"
							>
								<button
									type="button"
									class="flex min-h-14 min-w-0 flex-1 items-center gap-3 rounded-md px-2 text-left hover:bg-[var(--color-bg-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--app-accent)]"
									onclick={() => editTest(testCase)}
									aria-label={`${t('Open test')}: ${testCase.title}`}
								>
									<TestResult status={testCase.status} iconOnly /><span class="min-w-0 flex-1 truncate text-sm"
										>{testCase.title}</span
									><span class="hidden sm:block"><TestResult status={testCase.status} /></span>
								</button>
								{#if canEdit}<Menu.Root
										><Menu.Trigger
											disabled={saving}
											aria-label={`${t('Test actions')}: ${testCase.title}`}
											class="flex min-h-11 min-w-11 items-center justify-center rounded-md text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)]"
											><MoreHorizontal size={16} /></Menu.Trigger
										><Menu.Content align="end">
											<Menu.Item onclick={() => editTest(testCase)}>{t('Edit test')}</Menu.Item><Menu.Item
												disabled={plan.test_cases.length >= 200}
												onclick={() => duplicateTest(testCase)}>{t('Duplicate test')}</Menu.Item
											><Menu.Separator /><Menu.Item
												class="text-[var(--color-error)]"
												onclick={() => (plan.test_cases = plan.test_cases.filter((item) => item.id !== testCase.id))}
												>{t('Remove test')}</Menu.Item
											>
										</Menu.Content></Menu.Root
									>{/if}
							</div>{/each}
					</div>
				</section>
			</div>
		</form>
	{/if}
</div>
{#if editingTest}<TestCaseEditor
		testCase={editingTest}
		readonly={!canEdit || saving}
		onapply={applyTest}
		onclose={() => (editingTest = null)}
	/>{/if}
