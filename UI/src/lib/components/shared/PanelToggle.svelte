<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ChevronRight } from 'lucide-svelte';
	import { cn } from '$lib/utils';

	let {
		label,
		count,
		open = $bindable(false),
		children,
		class: className
	}: {
		label: string;
		count?: number;
		open?: boolean;
		children: Snippet;
		class?: string;
	} = $props();
	const id = $props.id();
</script>

<div class={cn('border-y border-[var(--app-border)]', className)} data-panel-toggle>
	<button
		type="button"
		aria-label={count === undefined ? label : `${label} (${count})`}
		aria-expanded={open}
		aria-controls={id}
		onclick={() => (open = !open)}
		class="flex min-h-11 w-full items-center gap-2 rounded-sm px-3 py-2 text-left text-sm font-medium text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-secondary)] hover:text-[var(--color-text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--app-accent)]"
	>
		<ChevronRight size={15} class={open ? 'rotate-90' : ''} aria-hidden="true" />
		<span class="min-w-0 break-words">{label}</span>
		{#if count !== undefined}<span class="shrink-0 text-xs font-normal tabular-nums text-[var(--color-text-tertiary)]"
				>({count})</span
			>{/if}
	</button>
	<div {id} hidden={!open}>
		{#if open}<div class="border-t border-[var(--app-border)]">{@render children()}</div>{/if}
	</div>
</div>
