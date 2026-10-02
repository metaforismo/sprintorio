<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { appToast } from '$lib/features/toast/toast';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, setLocale } from '$lib/paraglide/runtime.js';

	let {
		open = $bindable(false),
		onsubmit
	}: {
		open: boolean;
		onsubmit: (data: { name: string; key: string; description?: string }) => void | Promise<void>;
	} = $props();

	let submitting = $state(false);
	let name = $state('');
	let key = $state('');
	let description = $state('');
	let keyManuallyEdited = $state(false);

	$effect(() => {
		if (open) {
			name = '';
			key = '';
			description = '';
			keyManuallyEdited = false;
		}
	});

	// Auto-generate key from name (first 3 chars uppercase)
	$effect(() => {
		if (!keyManuallyEdited && name) {
			key = name
				.replace(/[^a-zA-Z]/g, '')
				.slice(0, 3)
				.toUpperCase();
		}
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (submitting || !name.trim() || !key.trim()) return;
		submitting = true;
		try {
			await onsubmit({
				name: name.trim(),
				key: key.trim().toUpperCase(),
				description: description.trim() || undefined
			});
			open = false;
		} catch (error) {
			appToast.apiError(error, m['sidebar.failed_create_team']());
		} finally {
			submitting = false;
		}
	}
</script>

<Dialog.Root
	{open}
	onOpenChange={(value) => {
		if (!submitting) open = value;
	}}
>
	<Dialog.Content
		showCloseButton={!submitting}
		class="sm:max-w-[420px] border-[var(--app-border)] bg-[var(--color-bg-secondary)] p-0 overflow-hidden rounded-xl"
	>
		<form onsubmit={handleSubmit} aria-busy={submitting}>
			<fieldset disabled={submitting} class="min-w-0">
				<div class="px-5 pt-5 pb-4 space-y-4">
					<div>
						<Dialog.Title class="text-base font-semibold text-[var(--color-text-primary)]"
							>{m['sidebar.create_team_title']()}</Dialog.Title
						>
						<Dialog.Description class="mt-0.5 text-xs text-[var(--color-text-tertiary)]"
							>{m['sidebar.create_team_desc']()}</Dialog.Description
						>
					</div>

					<div class="space-y-1.5">
						<Label for="create-name" class="text-xs text-[var(--color-text-secondary)]">{m['sidebar.name']()}</Label>
						<Input
							id="create-name"
							bind:value={name}
							maxlength={100}
							placeholder="e.g. Engineering"
							required
							class="bg-[var(--color-bg)] border-[var(--app-border)] text-[var(--color-text-primary)]"
						/>
					</div>

					<div class="space-y-1.5">
						<Label for="create-key" class="text-xs text-[var(--color-text-secondary)]"
							>{m['sidebar.identifier']()}</Label
						>
						<Input
							id="create-key"
							bind:value={key}
							placeholder="e.g. ENG"
							required
							maxlength={10}
							pattern="[A-Za-z]+"
							oninput={() => (keyManuallyEdited = true)}
							class="bg-[var(--color-bg)] border-[var(--app-border)] text-[var(--color-text-primary)] uppercase"
						/>
						<p class="text-[10px] text-[var(--color-text-tertiary)]">{m['sidebar.identifier_desc']()}</p>
					</div>

					<div class="space-y-1.5">
						<Label for="create-description" class="text-xs text-[var(--color-text-secondary)]"
							>{m['sidebar.description_optional']()}</Label
						>
						<Input
							id="create-description"
							bind:value={description}
							placeholder={m['sidebar.team_desc_placeholder']()}
							class="bg-[var(--color-bg)] border-[var(--app-border)] text-[var(--color-text-primary)]"
						/>
					</div>
				</div>

				<div class="flex justify-end gap-2 border-t border-[var(--app-border)] px-5 py-3">
					<Button variant="outline" size="sm" type="button" onclick={() => (open = false)}
						>{m['sidebar.cancel']()}</Button
					>
					<Button size="sm" type="submit" disabled={submitting || !name.trim() || !key.trim()}
						>{submitting ? m['sidebar.creating']() : m['sidebar.create_team_title']()}</Button
					>
				</div>
			</fieldset>
		</form>
	</Dialog.Content>
</Dialog.Root>
