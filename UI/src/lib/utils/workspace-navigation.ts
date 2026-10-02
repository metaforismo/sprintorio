export const LAST_WORKSPACE_KEY = 'sprintorio_last_workspace';

export function preferredWorkspace<T extends { slug: string }>(
	workspaces: T[],
	savedSlug: string | null
): T | undefined {
	return workspaces.find((workspace) => workspace.slug === savedSlug) ?? workspaces[0];
}

export function savedWorkspace(): string | null {
	try {
		return localStorage.getItem(LAST_WORKSPACE_KEY);
	} catch {
		return null;
	}
}

export function rememberWorkspace(slug: string): void {
	try {
		localStorage.setItem(LAST_WORKSPACE_KEY, slug);
	} catch {
		/* Navigation also works without storage. */
	}
}
