<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import { beforeNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { agentConfiguration, agentClients, type AgentClient } from '$lib/features/workspaces/agent-config';
	import { agentCopy } from '$lib/features/workspaces/agent-copy';
	import { listAgentTokens, createAgentToken, revokeAgentToken } from '$lib/api/agent-tokens';
	import type { AgentToken, AgentScope } from '$lib/types/agent-token';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import { KeyRound, Plus, Copy, Terminal, Bot } from 'lucide-svelte';
	const copy = $derived(agentCopy[getLocale() === 'it' ? 'it' : 'en']);
	const slug = $derived(page.params.workspaceSlug ?? '');
	let tokens = $state<AgentToken[]>([]);
	const activeTokens = $derived(tokens.filter(isActiveToken));
	const historyTokens = $derived(tokens.filter((token) => !isActiveToken(token)));
	let loading = $state(true);
	let loadError = $state(false);
	let createOpen = $state(false);
	let creating = $state(false);
	let name = $state('');
	let scope = $state<AgentScope>('read');
	let days = $state(30);
	let createError = $state(false);
	let secret = $state('');
	let secretOpen = $state(false);
	let revokeTarget = $state<AgentToken | null>(null);
	let revokeOpen = $state(false);
	let revoking = $state(false);
	let revokeError = $state(false);
	let clipboardStatus = $state('');
	let serverUrl = $state('');
	let mounted = true;
	let loadGeneration = 0;
	let navigationGeneration = 0;
	let client = $state<AgentClient>('claude');
	const config = $derived(agentConfiguration(client, serverUrl.trim() || page.url.origin, slug));
	const cliConfig = $derived(
		`export SPRINTORIO_URL=${shellQuote(serverUrl.trim() || page.url.origin)}\nexport SPRINTORIO_TOKEN='YOUR_TOKEN'\nexport SPRINTORIO_WORKSPACE=${shellQuote(slug)}\nsprintorio context`
	);
	function clearSecret() {
		secret = '';
		secretOpen = false;
		clipboardStatus = '';
	}
	beforeNavigate(() => {
		navigationGeneration++;
		clearSecret();
	});
	onDestroy(() => {
		mounted = false;
		clearSecret();
	});
	onMount(() => {
		serverUrl = window.location.origin;
	});
	$effect(() => {
		const workspaceSlug = slug;
		if (browser) {
			clearSecret();
			tokens = [];
			createOpen = false;
			revokeOpen = false;
			void load(workspaceSlug);
		}
	});
	async function load(workspaceSlug = slug) {
		const generation = ++loadGeneration;
		loading = true;
		loadError = false;
		try {
			const result = await listAgentTokens(workspaceSlug);
			if (generation === loadGeneration && mounted) tokens = result;
		} catch {
			if (generation === loadGeneration && mounted) loadError = true;
		} finally {
			if (generation === loadGeneration && mounted) loading = false;
		}
	}
	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (creating || !name.trim() || !Number.isInteger(days) || days < 1 || days > 365) return;
		const generation = navigationGeneration;
		const workspaceSlug = slug;
		creating = true;
		createError = false;
		try {
			const result = await createAgentToken(workspaceSlug, { name: name.trim(), scope, expires_in_days: days });
			if (!mounted || generation !== navigationGeneration || workspaceSlug !== slug) return;
			tokens = [result.record, ...tokens];
			createOpen = false;
			secret = result.token;
			secretOpen = true;
			clipboardStatus = '';
			name = '';
			scope = 'read';
			days = 30;
		} catch {
			createError = true;
		} finally {
			creating = false;
		}
	}
	async function revoke() {
		if (!revokeTarget || revoking) return;
		const workspaceSlug = slug;
		const generation = navigationGeneration;
		revoking = true;
		revokeError = false;
		try {
			const id = revokeTarget.id;
			await revokeAgentToken(workspaceSlug, id);
			if (!mounted || generation !== navigationGeneration || workspaceSlug !== slug) return;
			tokens = tokens.map((token) => (token.id === id ? { ...token, revoked_at: new Date().toISOString() } : token));
			revokeOpen = false;
			revokeTarget = null;
		} catch {
			revokeError = true;
		} finally {
			revoking = false;
		}
	}
	async function copyText(text: string) {
		clipboardStatus = '';
		try {
			await navigator.clipboard.writeText(text);
			clipboardStatus = copy.copied;
		} catch {
			clipboardStatus = copy.copyFailed;
		}
	}
	function shellQuote(value: string) {
		return "'" + value.replaceAll("'", "'\\''") + "'";
	}
	function date(value: string) {
		return new Intl.DateTimeFormat(getLocale(), { dateStyle: 'medium' }).format(new Date(value));
	}
	function isActiveToken(token: AgentToken) {
		return !token.revoked_at && new Date(token.expires_at).getTime() > Date.now();
	}
	function status(token: AgentToken) {
		return token.revoked_at
			? copy.revoked
			: new Date(token.expires_at).getTime() <= Date.now()
				? copy.expired
				: copy.active;
	}
</script>

{#snippet tokenRow(token: AgentToken)}
	<div class="flex flex-wrap items-center gap-3 p-4">
		<KeyRound size={18} class="shrink-0 text-[var(--color-text-tertiary)]" />
		<div class="min-w-0 flex-1">
			<div class="flex flex-wrap items-center gap-2">
				<h2 class="break-all text-sm font-medium">{token.name}</h2>
				<Badge variant="outline">{status(token)}</Badge>
			</div>
			<p class="mt-1 text-xs text-[var(--color-text-secondary)]">
				{token.scope === 'full' ? copy.full : token.scope === 'write' ? copy.write : copy.read} ·
				<code>{token.prefix}…</code>
			</p>
			<p class="mt-2 text-xs text-[var(--color-text-tertiary)]">
				{copy.expires}: {date(token.expires_at)} · {token.last_used_at
					? `${copy.used}: ${date(token.last_used_at)}`
					: copy.never}
			</p>
		</div>
		{#if isActiveToken(token)}<Button
				variant="outline"
				class="min-h-11"
				aria-label={`${copy.revoke} ${token.name}`}
				onclick={() => {
					revokeTarget = token;
					revokeError = false;
					revokeOpen = true;
				}}>{copy.revoke}</Button
			>{/if}
	</div>
{/snippet}

<svelte:head><title>{copy.title} · Sprintorio</title></svelte:head>
<div class="mx-auto max-w-3xl space-y-8 px-4 py-6 sm:px-8 sm:py-10">
	<header class="flex flex-wrap items-start justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold">{copy.title}</h1>
			<p class="mt-2 max-w-md text-sm text-[var(--color-text-secondary)]">{copy.detail}</p>
		</div>
		<Button
			class="min-h-11"
			onclick={() => {
				createError = false;
				createOpen = true;
			}}><Plus size={16} />{copy.create}</Button
		>
	</header>
	<section aria-label={copy.title} aria-busy={loading}>
		{#if loading}<p class="py-8 text-sm text-[var(--color-text-secondary)]" role="status">{copy.loading}</p>
		{:else if loadError}<div role="alert" class="rounded-xl border border-[var(--app-border)] p-5">
				<p>{copy.loadFailed}</p>
				<Button class="mt-3 min-h-11" variant="outline" onclick={() => load()}>{copy.retry}</Button>
			</div>
		{:else}
			{#if activeTokens.length === 0}
				<div
					class="rounded-xl border border-dashed border-[var(--app-border)] bg-[var(--color-bg-secondary)] px-5 py-6"
				>
					<Bot size={24} class="mb-3 text-[var(--color-text-tertiary)]" />
					<h2 class="font-medium">{tokens.length === 0 ? copy.empty : copy.noActive}</h2>
					<p class="mt-1 text-sm text-[var(--color-text-secondary)]">{copy.emptyDetail}</p>
				</div>
			{:else}
				<div
					class="divide-y divide-[var(--app-border)] rounded-xl border border-[var(--app-border)] bg-[var(--color-bg-secondary)]"
				>
					{#each activeTokens as token (token.id)}{@render tokenRow(token)}{/each}
				</div>
			{/if}
			{#if historyTokens.length > 0}
				<details
					class="mt-3 overflow-hidden rounded-xl border border-[var(--app-border)] bg-[var(--color-bg-secondary)]"
				>
					<summary class="min-h-11 cursor-pointer px-4 py-3 text-sm font-medium"
						>{copy.history} ({historyTokens.length})</summary
					>
					<div class="divide-y divide-[var(--app-border)] border-t border-[var(--app-border)]">
						{#each historyTokens as token (token.id)}{@render tokenRow(token)}{/each}
					</div>
				</details>
			{/if}
		{/if}
		<p class="mt-3 text-xs text-[var(--color-text-tertiary)]">{copy.personal}</p>
	</section>
	<section class="space-y-4 border-t border-[var(--app-border)] pt-6" aria-labelledby="connect-title">
		<div>
			<h2 id="connect-title" class="flex items-center gap-2 font-medium"><Terminal size={18} />{copy.guide}</h2>
			<p class="mt-2 text-sm text-[var(--color-text-secondary)]">{copy.guideHint}</p>
		</div>
		<details class="rounded-lg border border-[var(--app-border)] p-4">
			<summary class="cursor-pointer text-sm">{copy.build}</summary>
			<pre class="mt-3 overflow-x-auto rounded-lg bg-[var(--color-bg-secondary)] p-3 text-xs"><code
					>cd BE
go build -o sprintorio ./cmd/sprintorio
mkdir -p "$HOME/.local/bin"
install -m 755 sprintorio "$HOME/.local/bin/sprintorio"
export PATH="$HOME/.local/bin:$PATH"</code
				></pre>
		</details>
		<div class="space-y-2">
			<Label for="agent-server">{copy.endpoint}</Label><Input
				id="agent-server"
				class="min-h-11"
				bind:value={serverUrl}
				type="url"
			/>
			<p class="text-xs text-[var(--color-text-tertiary)]">{copy.endpointHint}</p>
		</div>

		<div class="space-y-2">
			<Label for="agent-client">{copy.client}</Label><select
				id="agent-client"
				bind:value={client}
				onchange={() => (clipboardStatus = '')}
				class="min-h-11 w-full rounded-lg border border-[var(--app-border)] bg-[var(--color-bg)] px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-[var(--app-accent)]"
				>{#each agentClients as option}<option value={option.id}>{option.name}</option>{/each}</select
			>
		</div>
		<div class="overflow-hidden rounded-xl border border-[var(--app-border)] bg-[var(--color-bg-secondary)]">
			<div class="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--app-border)] px-4 py-2">
				<h3 class="text-sm font-medium">{copy.config}</h3>
				<Button variant="ghost" class="min-h-11" onclick={() => copyText(config)}
					><Copy size={14} />{copy.copyConfig}</Button
				>
			</div>
			<pre class="overflow-x-auto p-4 text-xs"><code>{config}</code></pre>
		</div>
		<p class="text-xs text-[var(--color-text-secondary)]">{client === 'codex' ? copy.codexToken : copy.tokenHint}</p>
		<details class="rounded-lg border border-[var(--app-border)] p-4">
			<summary class="cursor-pointer text-sm">{copy.cli}</summary>
			<pre class="mt-3 overflow-x-auto text-xs"><code>{cliConfig}</code></pre>
		</details>
		<details class="rounded-lg border border-[var(--app-border)] p-4">
			<summary class="cursor-pointer text-sm">{copy.remote}</summary>
			<p class="mt-3 text-xs leading-relaxed text-[var(--color-text-secondary)]">{copy.remoteHint}</p>
			<pre class="mt-3 overflow-x-auto rounded-lg bg-[var(--color-bg-secondary)] p-3 text-xs"><code
					>sprintorio mcp-http --listen 127.0.0.1:8091 --origins https://client.example

https://mcp.example.com/mcp
Authorization: Bearer YOUR_TOKEN</code
				></pre>
		</details>
		{#if clipboardStatus && !secretOpen}<p role="status" class="text-xs">{clipboardStatus}</p>{/if}
	</section>
</div>

<Dialog.Root
	open={createOpen}
	onOpenChange={(open) => {
		if (!creating) createOpen = open;
	}}
>
	<Dialog.Content
		class="max-h-[80vh] overflow-y-auto bg-[var(--color-bg-secondary)]"
		showCloseButton={!creating}
		onEscapeKeydown={(event) => {
			if (creating) event.preventDefault();
		}}
		onInteractOutside={(event) => {
			if (creating) event.preventDefault();
		}}
	>
		<Dialog.Header
			><Dialog.Title>{copy.create}</Dialog.Title><Dialog.Description>{copy.emptyDetail}</Dialog.Description
			></Dialog.Header
		>
		<form onsubmit={create} aria-busy={creating} class="space-y-4">
			<div class="space-y-2">
				<Label for="agent-name">{copy.name}</Label><Input
					id="agent-name"
					class="min-h-11"
					bind:value={name}
					placeholder={copy.nameHint}
					maxlength={100}
					required
					disabled={creating}
				/>
			</div>
			<fieldset disabled={creating} class="space-y-2">
				<legend class="mb-2 text-sm font-medium">{copy.scope}</legend>
				<label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[var(--app-border)] p-3"
					><input class="mt-1" type="radio" name="agent-scope" value="read" bind:group={scope} /><span
						><span class="text-sm font-medium">{copy.read}</span><span
							class="mt-1 block text-xs text-[var(--color-text-secondary)]">{copy.readHint}</span
						></span
					></label
				>
				<label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[var(--app-border)] p-3"
					><input class="mt-1" type="radio" name="agent-scope" value="write" bind:group={scope} /><span
						><span class="text-sm font-medium">{copy.write}</span><span
							class="mt-1 block text-xs text-[var(--color-text-secondary)]">{copy.writeHint}</span
						></span
					></label
				>
				<label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[var(--app-border)] p-3"
					><input class="mt-1" type="radio" name="agent-scope" value="full" bind:group={scope} /><span
						><span class="text-sm font-medium">{copy.full}</span><span
							class="mt-1 block text-xs text-[var(--color-text-secondary)]">{copy.fullHint}</span
						></span
					></label
				>
			</fieldset>
			<div class="space-y-2">
				<Label for="agent-expiry">{copy.expiry}</Label><Input
					id="agent-expiry"
					class="min-h-11"
					type="number"
					min={1}
					max={365}
					step={1}
					bind:value={days}
					required
					disabled={creating}
				/>
			</div>
			{#if createError}<p role="alert" class="text-sm text-[var(--color-error)]">{copy.createFailed}</p>{/if}
			<Dialog.Footer
				><Button variant="outline" class="min-h-11" disabled={creating} onclick={() => (createOpen = false)}
					>{copy.cancel}</Button
				><Button
					class="min-h-11"
					type="submit"
					disabled={creating || !name.trim() || !Number.isInteger(days) || days < 1 || days > 365}
					>{creating ? copy.creating : copy.create}</Button
				></Dialog.Footer
			>
		</form>
	</Dialog.Content>
</Dialog.Root>
<Dialog.Root
	open={secretOpen}
	onOpenChange={(open) => {
		if (!open) clearSecret();
	}}
>
	<Dialog.Content class="bg-[var(--color-bg-secondary)]">
		<Dialog.Header
			><Dialog.Title>{copy.secretTitle}</Dialog.Title><Dialog.Description>{copy.secretHint}</Dialog.Description
			></Dialog.Header
		>
		<code
			class="select-all break-all rounded-lg border border-[var(--app-border)] bg-[var(--color-bg)] p-3 text-xs"
			data-agent-secret>{secret}</code
		>
		<Button class="min-h-11" onclick={() => copyText(secret)}><Copy size={16} />{copy.copyToken}</Button>
		{#if clipboardStatus}<p role="status" class="text-xs">{clipboardStatus}</p>{/if}
		<Dialog.Footer><Button variant="outline" class="min-h-11" onclick={clearSecret}>{copy.close}</Button></Dialog.Footer
		>
	</Dialog.Content>
</Dialog.Root>
<Dialog.Root
	open={revokeOpen}
	onOpenChange={(open) => {
		if (!revoking) revokeOpen = open;
	}}
>
	<Dialog.Content
		class="bg-[var(--color-bg-secondary)]"
		showCloseButton={!revoking}
		onEscapeKeydown={(event) => {
			if (revoking) event.preventDefault();
		}}
		onInteractOutside={(event) => {
			if (revoking) event.preventDefault();
		}}
	>
		<Dialog.Header
			><Dialog.Title>{copy.revokeTitle}</Dialog.Title><Dialog.Description
				>{revokeTarget?.name} · {copy.revokeHint}</Dialog.Description
			></Dialog.Header
		>
		{#if revokeError}<p role="alert" class="text-sm text-[var(--color-error)]">{copy.revokeFailed}</p>{/if}
		<Dialog.Footer
			><Button variant="outline" class="min-h-11" disabled={revoking} onclick={() => (revokeOpen = false)}
				>{copy.cancel}</Button
			><Button variant="destructive" class="min-h-11" disabled={revoking} onclick={revoke}
				>{revoking ? copy.revoking : copy.revoke}</Button
			></Dialog.Footer
		>
	</Dialog.Content>
</Dialog.Root>
