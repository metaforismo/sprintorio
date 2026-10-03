<script lang="ts">
	import { untrack } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { deliveryText } from './delivery-copy';
	import type { DeliveryPlan, TestStatus } from '$lib/types/delivery-plan';
	import ChoiceSelect from './ChoiceSelect.svelte';
	type TestCase = DeliveryPlan['test_cases'][number];
	let {
		testCase,
		readonly = false,
		onapply,
		onclose
	}: { testCase: TestCase; readonly?: boolean; onapply: (test: TestCase) => void; onclose: () => void } = $props();
	let draft = $state<TestCase>(untrack(() => JSON.parse(JSON.stringify(testCase))));
	const t = (text: string) => deliveryText(text, getLocale());
	function invalidate() {
		draft.status = 'not_run';
		draft.evidence = '';
	}
	function apply(event: SubmitEvent) {
		event.preventDefault();
		if (!readonly) onapply(JSON.parse(JSON.stringify(draft)));
	}
</script>

<Dialog.Root open onOpenChange={(open) => !open && onclose()}>
	<Dialog.Content animate={false} class="max-h-[85dvh] overflow-y-auto sm:max-w-2xl">
		<Dialog.Header><Dialog.Title>{readonly ? t('Test details') : t('Edit test')}</Dialog.Title></Dialog.Header>
		<form onsubmit={apply} class="space-y-5">
			<div class="space-y-1.5">
				<label for="delivery-test-title" class="text-sm">{t('Test case')}</label><Input
					id="delivery-test-title"
					class="min-h-11"
					required
					maxlength={300}
					disabled={readonly}
					bind:value={draft.title}
					oninput={invalidate}
				/>
			</div>
			<div class="grid gap-4 sm:grid-cols-2">
				<div class="space-y-1.5">
					<label for="delivery-test-steps" class="text-sm">{t('Steps')}</label><Textarea
						id="delivery-test-steps"
						rows={4}
						maxlength={10000}
						disabled={readonly}
						bind:value={draft.steps}
						oninput={invalidate}
					/>
				</div>
				<div class="space-y-1.5">
					<label for="delivery-test-expected" class="text-sm">{t('Expected result')}</label><Textarea
						id="delivery-test-expected"
						rows={4}
						maxlength={5000}
						disabled={readonly}
						bind:value={draft.expected_result}
						oninput={invalidate}
					/>
				</div>
			</div>
			<div class="space-y-1.5">
				<span class="text-sm">{t('Test status')}</span><ChoiceSelect
					label={t('Test status')}
					value={draft.status}
					disabled={readonly}
					options={[
						{ value: 'not_run', label: t('Not run') },
						{ value: 'passed', label: t('Passed') },
						{ value: 'failed', label: t('Failed') },
						{ value: 'blocked', label: t('Blocked') }
					]}
					onchange={(value) => (draft.status = value as TestStatus)}
				/>
			</div>
			<div class="space-y-1.5">
				<label for="delivery-test-evidence" class="text-sm"
					>{t('Evidence')}{draft.status === 'passed' || draft.status === 'failed' ? ' *' : ''}</label
				><Textarea
					id="delivery-test-evidence"
					rows={3}
					required={draft.status === 'passed' || draft.status === 'failed'}
					maxlength={10000}
					disabled={readonly}
					bind:value={draft.evidence}
					placeholder={t('Observed result, environment, and links to proof')}
				/>
			</div>
			<Dialog.Footer
				><Button class="max-md:min-h-11" variant="ghost" onclick={onclose}>{readonly ? t('Close') : t('Cancel')}</Button>{#if !readonly}<Button
						class="max-md:min-h-11" type="submit">{t('Apply changes')}</Button
					>{/if}</Dialog.Footer
			>
		</form>
	</Dialog.Content>
</Dialog.Root>
