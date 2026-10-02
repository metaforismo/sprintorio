<script lang="ts">
	import type { Team } from '$lib/types/team';
	import { getStableTeamColor } from '$lib/features/layout/sidebar.state.svelte';
	import { Box, CircleDot, Layers, Settings, ShieldCheck, SquareUser, Users } from 'lucide-svelte';
	import { loadTeamIcon, type TeamIconComponent } from '$lib/utils/team-icons';

	const LEGACY_ICON_NAMES: Record<string, string> = {
		box: 'Box',
		'circle-dot': 'CircleDot',
		layers: 'Layers',
		settings: 'Settings',
		shield: 'ShieldCheck',
		'square-user': 'SquareUser',
		users: 'Users'
	};
	const commonIcons: Record<string, TeamIconComponent> = { Box, CircleDot, Layers, Settings, ShieldCheck, SquareUser, Users };

	function resolveIconName(icon?: string | null): string {
		if (!icon || icon.startsWith('emoji:')) return 'SquareUser';
		return LEGACY_ICON_NAMES[icon] ?? icon;
	}

	let {
		team,
		size = 16,
		class: className = 'shrink-0'
	}: {
		team: Team;
		size?: number;
		class?: string;
	} = $props();

	let customIcon = $state<TeamIconComponent | null>(null);
	const iconName = $derived(resolveIconName(team.icon));
	const Icon = $derived(commonIcons[iconName] ?? customIcon ?? SquareUser);
	const emoji = $derived(team.icon?.startsWith('emoji:') ? team.icon.slice(6) : null);
	const color = $derived(getStableTeamColor(team));

	$effect(() => {
		customIcon = null;
		if (emoji || commonIcons[iconName]) return;
		let active = true;
		loadTeamIcon(iconName).then((icon) => {
			if (active) customIcon = icon;
		});
		return () => { active = false; };
	});
</script>

{#if emoji}
	<span class={className} style="font-size: {size}px; line-height: 1" aria-hidden="true">{emoji}</span>
{:else}
	<Icon {size} class={className} style="color: {color}" />
{/if}
