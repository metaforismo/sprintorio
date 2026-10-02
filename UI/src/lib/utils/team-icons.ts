import type { SquareUser } from 'lucide-svelte';

export type TeamIconComponent = typeof SquareUser;

let catalog: Promise<Record<string, TeamIconComponent>> | undefined;

export async function loadTeamIcon(name: string): Promise<TeamIconComponent | null> {
	try {
		catalog ??= import('lucide-svelte').then((icons) => icons as unknown as Record<string, TeamIconComponent>);
		const icons = await catalog;
		const icon = icons[name];
		return typeof icon === 'function' ? icon : null;
	} catch {
		catalog = undefined;
		return null;
	}
}
