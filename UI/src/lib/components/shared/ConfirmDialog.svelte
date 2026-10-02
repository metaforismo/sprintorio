<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, setLocale } from '$lib/paraglide/runtime.js';

	let {
		title,
		message,
		confirmLabel,
		variant = 'danger',
		onconfirm,
		oncancel
	}: {
		title: string;
		message: string;
		confirmLabel?: string;
		variant?: 'danger' | 'default';
		onconfirm: () => void;
		oncancel: () => void;
	} = $props();

	const effectiveConfirmLabel = $derived(confirmLabel ?? m['sharedComponents.confirm_dialog.confirm']());
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center">
	<div
		role="button"
		tabindex="0"
		aria-label={m['sharedComponents.confirm_dialog.cancel']()}
		class="fixed inset-0 bg-black/50"
		onclick={oncancel}
		onkeydown={(e) => { if (e.key === 'Enter' || e.key === 'Escape') oncancel(); }}
	></div>
	<div
		class="relative z-10 w-full max-w-sm rounded-lg border border-[var(--app-border)] bg-[var(--color-bg-secondary)] p-6 shadow-2xl"
	>
		<h3 class="text-sm font-medium text-[var(--color-text-primary)]">{title}</h3>
		<p class="mt-2 text-sm text-[var(--color-text-secondary)]">{message}</p>
		<div class="mt-4 flex justify-end gap-2">
			<button
				onclick={oncancel}
				class="rounded-md border border-[var(--app-border)] px-3 py-1.5 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]"
			>
				{m['sharedComponents.confirm_dialog.cancel']()}
			</button>
			<button
				onclick={onconfirm}
				class="rounded-md px-3 py-1.5 text-sm text-white {variant === 'danger'
					? 'bg-[var(--color-error)] hover:bg-red-600'
					: 'bg-[var(--app-accent)] hover:bg-[var(--app-accent-hover)]'}"
			>
				{effectiveConfirmLabel}
			</button>
		</div>
	</div>
</div>
