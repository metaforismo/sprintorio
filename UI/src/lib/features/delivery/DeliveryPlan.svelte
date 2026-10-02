<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { getDeliveryPlan, saveDeliveryPlan } from '$lib/api/delivery-plan';
	import { getWorkspace } from '$lib/api/workspaces';
	import { hasPermission } from '$lib/security/permissions';
	import type { Role } from '$lib/security/roles';
	import { emptyDeliveryPlan, type DeliveryPlan } from '$lib/types/delivery-plan';
	import { Button } from '$lib/components/ui/button';
	import { Plus, Trash2, CheckCircle2, ClipboardCheck, Flag, Package } from 'lucide-svelte';

	let { slug, projectId }: { slug: string; projectId: string } = $props();
	let plan = $state<DeliveryPlan>(emptyDeliveryPlan());
	let saved = $state('');
	let version = $state(0);
	let loading = $state(true);
	let saving = $state(false);
	let canEdit = $state(false);
	let error = $state('');
	let conflict = $state(false);
	let notice = $state('');
	let generation = 0;
	const dirty = $derived(saved !== '' && JSON.stringify(plan) !== saved);
	const passed = $derived(plan.test_cases.filter((t) => t.status === 'passed').length);
	const failed = $derived(plan.test_cases.filter((t) => t.status === 'failed').length);
	const blocked = $derived(plan.test_cases.filter((t) => t.status === 'blocked').length);
	const pending = $derived(plan.test_cases.filter((t) => t.status === 'not_run').length);
	const completed = $derived(plan.milestones.filter((m) => m.status === 'done').length);
	const briefComplete = $derived(!!(plan.product_name.trim() && plan.objective.trim() && plan.success_metric.trim()));
	const testDefinitionsComplete = $derived(
		plan.test_cases.every((t) => !!(t.title.trim() && t.steps.trim() && t.expected_result.trim() && t.evidence.trim()))
	);
	const readiness = $derived(
		failed
			? 'Changes needed'
			: blocked
				? 'Testing blocked'
				: !plan.test_cases.length
					? 'No tests recorded'
					: pending
						? 'Testing pending'
						: !briefComplete || !testDefinitionsComplete || completed < plan.milestones.length
							? 'Delivery work pending'
							: 'Ready for review'
	);
	const fieldClass =
		'w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg-primary)] px-3 py-2 text-sm text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] disabled:opacity-70';

	async function load(s = slug, id = projectId, request = ++generation) {
		loading = true;
		error = '';
		notice = '';
		try {
			const [result, workspace] = await Promise.all([getDeliveryPlan(s, id), getWorkspace(s)]);
			if (request !== generation || s !== slug || id !== projectId) return;
			plan = result.plan;
			version = result.version;
			saved = JSON.stringify(result.plan);
			canEdit = hasPermission(workspace.current_user_role as Role, 'project:manage');
			conflict = false;
		} catch {
			if (request === generation) error = 'Could not load the delivery plan. Try again.';
		} finally {
			if (request === generation) loading = false;
		}
	}
	$effect(() => {
		const s = slug;
		const id = projectId;
		const request = ++generation;
		plan = emptyDeliveryPlan();
		saved = '';
		canEdit = false;
		saving = false;
		conflict = false;
		void load(s, id, request);
		return () => {
			if (generation === request) generation++;
		};
	});
	beforeNavigate(({ cancel }) => {
		if (dirty && !window.confirm('Leave this project and discard unsaved delivery changes?')) cancel();
	});
	function preventUnload(event: BeforeUnloadEvent) {
		if (dirty) {
			event.preventDefault();
			event.returnValue = '';
		}
	}
	function cancelChanges() {
		plan = JSON.parse(saved);
		error = conflict ? 'Someone saved a newer plan. Reload the latest version before editing again.' : '';
		notice = '';
	}
	function addMilestone() {
		plan.milestones.push({ id: crypto.randomUUID(), title: '', due_date: '', status: 'planned' });
		notice = '';
	}
	function invalidateTest(testCase: DeliveryPlan['test_cases'][number]) {
		if (testCase.status !== 'not_run') {
			testCase.status = 'not_run';
			testCase.evidence = '';
		}
	}
	function addTest() {
		plan.test_cases.push({
			id: crypto.randomUUID(),
			title: '',
			steps: '',
			expected_result: '',
			status: 'not_run',
			evidence: ''
		});
		notice = '';
	}
	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (!canEdit || saving || !dirty || conflict) return;
		const s = slug;
		const id = projectId;
		const request = generation;
		const snapshot = JSON.parse(JSON.stringify(plan)) as DeliveryPlan;
		if (snapshot.milestones.some((m) => !m.title.trim()) || snapshot.test_cases.some((t) => !t.title.trim())) {
			error = 'Give every milestone and test case a title before saving.';
			return;
		}
		if (snapshot.test_cases.some((t) => (t.status === 'passed' || t.status === 'failed') && !t.evidence.trim())) {
			error = 'Passed and failed tests need recorded evidence before saving.';
			return;
		}
		saving = true;
		error = '';
		notice = '';
		try {
			const result = await saveDeliveryPlan(s, id, snapshot, version);
			if (request !== generation || s !== slug || id !== projectId) return;
			plan = result.plan;
			saved = JSON.stringify(result.plan);
			version = result.version;
			notice = 'Delivery plan saved.';
		} catch (err) {
			if (request !== generation) return;
			const code = (err as { error?: { code?: string } })?.error?.code;
			conflict = !!code && /conflict/i.test(code);
			error = conflict
				? 'Someone saved a newer plan. Your edits are still here. Copy anything you want to keep before reloading.'
				: 'Could not save the delivery plan. Your edits are still here. Try again.';
		} finally {
			if (request === generation) saving = false;
		}
	}
</script>

<svelte:window onbeforeunload={preventUnload} />
<div class="mx-auto w-full max-w-5xl px-4 py-6 sm:px-8">
	{#if loading}
		<p role="status" class="py-12 text-center text-sm text-[var(--color-text-tertiary)]">Loading delivery plan…</p>
	{:else if saved === ''}
		<div role="alert" class="rounded-lg border border-[var(--app-border)] p-6">
			<p class="mb-4">{error}</p>
			<Button variant="outline" onclick={() => load()}>Retry loading</Button>
		</div>
	{:else}
		<form onsubmit={save} class="space-y-6">
			<div class="flex flex-wrap items-start justify-between gap-4">
				<div>
					<h1 class="text-xl font-semibold">Product delivery</h1>
					<p class="mt-1 text-sm text-[var(--color-text-tertiary)]">
						Define the outcome, track milestones, and record manual test results.
					</p>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					{#if dirty}<span class="text-xs text-[var(--color-text-tertiary)]">Unsaved changes</span>{/if}
					{#if canEdit}<Button type="button" variant="outline" disabled={!dirty || saving} onclick={cancelChanges}
							>Cancel changes</Button
						><Button type="submit" disabled={!dirty || saving || conflict}>{saving ? 'Saving…' : 'Save plan'}</Button
						>{/if}
				</div>
			</div>
			{#if !canEdit}<p class="text-sm text-[var(--color-text-tertiary)]">
					View only. Workspace owners, admins, and members can edit this plan.
				</p>{/if}
			{#if error}<div role="alert" class="rounded-lg border border-[var(--color-error)] p-4 text-sm">
					<p>{error}</p>
					{#if conflict}<Button type="button" variant="outline" class="mt-3" onclick={() => load()}
							>Reload latest and discard my edits</Button
						>{/if}
				</div>{/if}
			{#if notice && !dirty}<p role="status" class="text-sm text-[var(--color-text-secondary)]">{notice}</p>{/if}
			<div class="rounded-xl border border-[var(--app-border)] bg-[var(--color-bg-secondary)] p-5">
				<div class="flex flex-wrap items-center gap-2 font-medium">
					<ClipboardCheck size={18} /><span>{readiness}</span>{#if dirty}<span
							class="text-xs font-normal text-[var(--color-text-tertiary)]">Based on unsaved changes</span
						>{/if}
				</div>
				<div class="mt-4 grid grid-cols-2 gap-4 text-sm sm:grid-cols-5">
					<div>
						<p class="text-lg font-semibold">{completed}/{plan.milestones.length}</p>
						<p class="text-[var(--color-text-tertiary)]">Milestones done</p>
					</div>
					<div>
						<p class="text-lg font-semibold">{passed}</p>
						<p class="text-[var(--color-text-tertiary)]">Passed</p>
					</div>
					<div>
						<p class="text-lg font-semibold">{failed}</p>
						<p class="text-[var(--color-text-tertiary)]">Failed</p>
					</div>
					<div>
						<p class="text-lg font-semibold">{blocked}</p>
						<p class="text-[var(--color-text-tertiary)]">Blocked</p>
					</div>
					<div>
						<p class="text-lg font-semibold">{pending}</p>
						<p class="text-[var(--color-text-tertiary)]">Not run</p>
					</div>
				</div>
				<p class="mt-4 text-xs text-[var(--color-text-tertiary)]">
					Results are recorded manually. Ready for review means the brief is complete, listed milestones are done, and
					all tests have steps, expected results, and recorded evidence with every test passed. Release approval is a
					separate decision.
				</p>
			</div>
			<fieldset disabled={!canEdit || saving} class="space-y-6">
				<section class="rounded-xl border border-[var(--app-border)] p-5">
					<h2 class="mb-5 flex items-center gap-2 font-medium"><Package size={18} />Product brief</h2>
					<div class="grid gap-4 sm:grid-cols-2">
						<label class="space-y-1.5 text-sm"
							><span>Product name</span><input
								class={fieldClass}
								maxlength="200"
								bind:value={plan.product_name}
								placeholder="What are you delivering?"
							/></label
						>
						<label class="space-y-1.5 text-sm"
							><span>Target release</span><input
								type="date"
								class={fieldClass}
								bind:value={plan.target_release}
							/></label
						>
						<label class="space-y-1.5 text-sm sm:col-span-2"
							><span>Objective</span><textarea
								rows="3"
								class={fieldClass}
								maxlength="5000"
								bind:value={plan.objective}
								placeholder="The problem this release should solve"
							></textarea></label
						>
						<label class="space-y-1.5 text-sm sm:col-span-2"
							><span>Success metric</span><textarea
								rows="2"
								class={fieldClass}
								maxlength="2000"
								bind:value={plan.success_metric}
								placeholder="How will you know it worked?"
							></textarea></label
						>
					</div>
				</section>
				<section class="rounded-xl border border-[var(--app-border)] p-5">
					<div class="mb-4 flex items-center justify-between gap-3">
						<h2 class="flex flex-wrap items-center gap-2 font-medium"><Flag size={18} />Milestones</h2>
						{#if canEdit}<Button
								type="button"
								variant="outline"
								size="sm"
								disabled={plan.milestones.length >= 200}
								onclick={addMilestone}><Plus size={14} />Add milestone</Button
							>{/if}
					</div>
					{#if !plan.milestones.length}<p class="py-4 text-sm text-[var(--color-text-tertiary)]">
							No milestones yet. Add the checkpoints needed to deliver this product.
						</p>{/if}
					<div class="space-y-3">
						{#each plan.milestones as milestone, i (milestone.id)}
							<div
								class="grid items-end gap-3 rounded-lg bg-[var(--color-bg-secondary)] p-4 sm:grid-cols-2 lg:grid-cols-[1fr_150px_150px_auto]"
							>
								<label class="space-y-1 text-sm"
									><span>Milestone {i + 1}</span><input
										class={fieldClass}
										required
										maxlength="300"
										bind:value={milestone.title}
									/></label
								>
								<label class="space-y-1 text-sm"
									><span>Due date</span><input type="date" class={fieldClass} bind:value={milestone.due_date} /></label
								>
								<label class="space-y-1 text-sm"
									><span>Milestone status</span><select class={fieldClass} bind:value={milestone.status}
										><option value="planned">Planned</option><option value="in_progress">In progress</option><option
											value="done">Done</option
										></select
									></label
								>
								{#if canEdit}<Button
										type="button"
										variant="ghost"
										aria-label={`Remove milestone ${i + 1}`}
										onclick={() => (plan.milestones = plan.milestones.filter((m) => m.id !== milestone.id))}
										><Trash2 size={16} /></Button
									>{/if}
							</div>
						{/each}
					</div>
				</section>
				<section class="rounded-xl border border-[var(--app-border)] p-5">
					<div class="mb-1 flex items-center justify-between gap-3">
						<h2 class="flex flex-wrap items-center gap-2 font-medium"><CheckCircle2 size={18} />Manual tests</h2>
						{#if canEdit}<Button
								type="button"
								variant="outline"
								size="sm"
								disabled={plan.test_cases.length >= 200}
								onclick={addTest}><Plus size={14} />Add test case</Button
							>{/if}
					</div>
					<p class="mb-5 text-sm text-[var(--color-text-tertiary)]">
						Run these checks yourself, then record the result and evidence.
					</p>
					{#if !plan.test_cases.length}<p class="py-4 text-sm text-[var(--color-text-tertiary)]">
							No test cases yet. Delivery readiness cannot be assessed without recorded tests.
						</p>{/if}
					<div class="space-y-4">
						{#each plan.test_cases as testCase, i (testCase.id)}
							<div class="space-y-4 rounded-lg bg-[var(--color-bg-secondary)] p-4">
								<div class="flex items-end gap-3">
									<label class="min-w-0 flex-1 space-y-1 text-sm"
										><span>Test case {i + 1}</span><input
											class={fieldClass}
											required
											maxlength="300"
											bind:value={testCase.title}
											oninput={() => invalidateTest(testCase)}
										/></label
									>{#if canEdit}<Button
											type="button"
											variant="ghost"
											aria-label={`Remove test case ${i + 1}`}
											onclick={() => (plan.test_cases = plan.test_cases.filter((t) => t.id !== testCase.id))}
											><Trash2 size={16} /></Button
										>{/if}
								</div>
								<div class="grid gap-4 sm:grid-cols-2">
									<label class="space-y-1 text-sm"
										><span>Steps</span><textarea
											rows="3"
											maxlength="10000"
											class={fieldClass}
											bind:value={testCase.steps}
											oninput={() => invalidateTest(testCase)}
											placeholder="Actions to perform"
										></textarea></label
									><label class="space-y-1 text-sm"
										><span>Expected result</span><textarea
											rows="3"
											maxlength="5000"
											class={fieldClass}
											bind:value={testCase.expected_result}
											oninput={() => invalidateTest(testCase)}
											placeholder="What should happen"
										></textarea></label
									>
								</div>
								<div class="grid gap-4 sm:grid-cols-[180px_1fr]">
									<label class="space-y-1 text-sm"
										><span>Test status</span><select class={fieldClass} bind:value={testCase.status}
											><option value="not_run">Not run</option><option value="passed">Passed</option><option
												value="failed">Failed</option
											><option value="blocked">Blocked</option></select
										></label
									><label class="space-y-1 text-sm"
										><span
											>Evidence{testCase.status === 'passed' || testCase.status === 'failed' ? ' (required)' : ''}</span
										><textarea
											rows="2"
											maxlength="10000"
											required={testCase.status === 'passed' || testCase.status === 'failed'}
											class={fieldClass}
											bind:value={testCase.evidence}
											placeholder="Observed result, environment, and links to proof"
										></textarea></label
									>
								</div>
							</div>
						{/each}
					</div>
				</section>
			</fieldset>
		</form>
	{/if}
</div>
