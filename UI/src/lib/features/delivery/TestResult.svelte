<script lang="ts">
	import { CircleDashed, CheckCircle2, XCircle, CircleSlash } from 'lucide-svelte';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { deliveryText } from './delivery-copy';
	import type { TestStatus } from '$lib/types/delivery-plan';
	let { status, iconOnly = false }: { status: TestStatus; iconOnly?: boolean } = $props();
	const label = $derived(
		deliveryText(
			status === 'passed' ? 'Passed' : status === 'failed' ? 'Failed' : status === 'blocked' ? 'Blocked' : 'Not run',
			getLocale()
		)
	);
	const color = $derived(
		status === 'passed'
			? 'var(--color-success, #6bb58a)'
			: status === 'failed'
				? 'var(--color-error)'
				: status === 'blocked'
					? 'var(--color-warning, #c7a96b)'
					: 'var(--color-text-tertiary)'
	);
</script>

<span class="inline-flex shrink-0 items-center gap-1.5 text-xs" style:color aria-label={iconOnly ? label : undefined}>
	{#if iconOnly}{#if status === 'passed'}<CheckCircle2 size={16} />{:else if status === 'failed'}<XCircle
				size={16}
			/>{:else if status === 'blocked'}<CircleSlash size={16} />{:else}<CircleDashed
				size={16}
			/>{/if}{:else}{label}{/if}
</span>
