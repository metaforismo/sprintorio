<script lang="ts">
	import * as Popover from '$lib/components/ui/popover';
	import { RangeCalendar } from '$lib/components/ui/range-calendar';
	import { CalendarDate } from '@internationalized/date';
	import type { DateValue } from '@internationalized/date';
	import { CalendarIcon, X } from 'lucide-svelte';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let {
		startDate = null,
		endDate = null,
		onchange,
		isDateDisabled,
		numberOfMonths = 2,
		placeholder = 'Select dates'
	}: {
		startDate: string | null;
		endDate: string | null;
		onchange: (start: string, end: string) => void;
		isDateDisabled?: (date: DateValue) => boolean;
		numberOfMonths?: number;
		placeholder?: string;
	} = $props();

	let open = $state(false);
	let triggerElement: HTMLButtonElement | undefined = $state();

	const calendarValue = $derived.by(() => {
		const start = startDate ? parseDate(startDate) : undefined;
		const end = endDate ? parseDate(endDate) : undefined;
		if (!start && !end) return undefined;
		return { start, end };
	});

	function parseDate(value: string): CalendarDate | undefined {
		try {
			const d = new Date(value);
			return new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
		} catch {
			return undefined;
		}
	}

	function formatDate(value: string): string {
		try {
			return new Date(value).toLocaleDateString(getLocale(), {
				month: 'short',
				day: 'numeric',
				year: 'numeric'
			});
		} catch {
			return value;
		}
	}

	const displayText = $derived.by(() => {
		if (startDate && endDate) {
			return `${formatDate(startDate)} – ${formatDate(endDate)}`;
		}
		if (startDate) return formatDate(startDate);
		return null;
	});

	function handleValueChange(range: { start: DateValue | undefined; end: DateValue | undefined } | undefined) {
		if (range?.start && range?.end) {
			const start = `${range.start.year}-${String(range.start.month).padStart(2, '0')}-${String(range.start.day).padStart(2, '0')}`;
			const end = `${range.end.year}-${String(range.end.month).padStart(2, '0')}-${String(range.end.day).padStart(2, '0')}`;
			onchange(start, end);
			open = false;
		}
	}

	function handleClear() {
		onchange('', '');
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
					class="flex items-center gap-1.5 rounded-md border border-[var(--app-border)] bg-[var(--color-bg-secondary)] px-2.5 py-1 text-xs hover:bg-[var(--color-bg-hover)] {displayText
						? 'text-[var(--color-text-primary)]'
						: 'text-[var(--color-text-secondary)]'}"
				>
					<CalendarIcon size={12} />
					{#if displayText}
						{displayText}
					{:else}
						{placeholder}
					{/if}
				</button>
			{/snippet}
		</Popover.Trigger>
		{#if displayText}
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
		{#key `${startDate}-${endDate}`}
			<RangeCalendar value={calendarValue} onValueChange={handleValueChange} {numberOfMonths} {isDateDisabled} />
		{/key}
	</Popover.Content>
</Popover.Root>
