<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Info } from 'lucide-svelte';
	import * as Popover from '$lib/components/ui/popover';
	import { cn } from '$lib/utils';

	let {
		label,
		title,
		children,
		align = 'end',
		class: className
	}: {
		label: string;
		title?: string;
		children: Snippet;
		align?: 'start' | 'center' | 'end';
		class?: string;
	} = $props();
</script>

<Popover.Root>
	<Popover.Trigger
		aria-label={label}
		class={cn(
			'inline-flex size-8 max-md:size-11 shrink-0 items-center justify-center rounded-md text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-tertiary)] hover:text-[var(--color-text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--app-accent)] active:scale-[0.97]',
			className
		)}
	>
		<Info size={15} aria-hidden="true" />
	</Popover.Trigger>
	<Popover.Content
		{align}
		sideOffset={6}
		class="w-80 max-w-[calc(100vw-2rem)] gap-2 border border-[var(--app-border)] bg-[var(--color-bg-secondary)] p-4 text-xs leading-relaxed text-[var(--color-text-secondary)] shadow-lg"
	>
		{#if title}<Popover.Title class="text-sm font-medium text-[var(--color-text-primary)]">{title}</Popover.Title>{/if}
		<div class="min-w-0 break-words">{@render children()}</div>
	</Popover.Content>
</Popover.Root>
