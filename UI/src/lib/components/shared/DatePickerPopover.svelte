<script lang="ts">
	import * as Popover from '$lib/components/ui/popover';
	import { Calendar } from '$lib/components/ui/calendar';
	import DueDatePickerPanel from './DueDatePickerPanel.svelte';
	import { CalendarDate } from '@internationalized/date';
	import type { DateValue } from '@internationalized/date';
	import { CalendarIcon, X } from 'lucide-svelte';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let {
		value = null,
		onchange,
		placeholder = 'Set date',
		colorClass = '',
		dueDateMode = false
	}: {
		value: string | null;
		onchange: (date: string | null) => void;
		placeholder?: string;
		colorClass?: string;
		dueDateMode?: boolean;
	} = $props();

	let open = $state(false);
	let triggerElement: HTMLButtonElement | undefined = $state();

	const displayDate = $derived.by(() => {
		if (!value) return null;
		try {
			return new Date(value).toLocaleDateString(getLocale(), {
				month: 'short',
				day: 'numeric',
				year: 'numeric'
			});
		} catch {
			return null;
		}
	});

	const calendarValue = $derived.by(() => {
		if (!value) return undefined;
		try {
			const d = new Date(value);
			return new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
		} catch {
			return undefined;
		}
	});

	function handleSelect(dateValue: DateValue | undefined) {
		if (dateValue) {
			const date = `${dateValue.year}-${String(dateValue.month).padStart(2, '0')}-${String(dateValue.day).padStart(2, '0')}`;
			onchange(date);
			open = false;
		}
	}

	function handleClear() {
		onchange(null);
		open = false;
		triggerElement?.focus();
	}
</script>

<Popover.Root bind:open>
	<div class="inline-flex items-center gap-1">
		<Popover.Trigger>
			{#snippet child({ props })}
				<button
					{...props}
					bind:this={triggerElement}
					type="button"
					class="flex items-center gap-1.5 rounded-md border border-[var(--app-border)] bg-[var(--color-bg-secondary)] px-2.5 py-1 text-xs hover:bg-[var(--color-bg-hover)] {displayDate &&
					colorClass
						? colorClass
						: 'text-[var(--color-text-secondary)]'}"
				>
					<CalendarIcon size={12} />
					{#if displayDate}
						{displayDate}
					{:else}
						{placeholder}
					{/if}
				</button>
			{/snippet}
		</Popover.Trigger>
		{#if displayDate}
			<button
				type="button"
				onclick={handleClear}
				aria-label={getLocale() === 'it' ? 'Rimuovi date' : 'Clear date'}
				class="inline-flex min-h-7 min-w-7 items-center justify-center rounded-md text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]"
			>
				<X size={12} />
			</button>
		{/if}
	</div>
	<Popover.Content class="w-auto p-0" align="start">
		{#if dueDateMode}
			<DueDatePickerPanel {value} {onchange} close={() => (open = false)} />
		{:else}
			{#key value}
				<Calendar type="single" value={calendarValue} onValueChange={handleSelect} />
			{/key}
		{/if}
	</Popover.Content>
</Popover.Root>
