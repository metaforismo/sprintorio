/** Resolve only releases published for Sprintorio. The default state is unreleased;
 * a missing or empty manifest never implies that a release has been published.
 */

export const FALLBACK_VERSION = 'Unreleased';
export const RELEASES_PAGE_URL = 'https://github.com/metaforismo/sprintorio/releases';

const RELEASES_MANIFEST_URL =
	'https://raw.githubusercontent.com/metaforismo/sprintorio/main/UI/static/releases.json';
const RELEASE_VERSION = /^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;
const RELEASE_RETRY_DELAYS_MS = [0, 250, 1000];

interface GitHubRelease {
	tag_name: string;
	html_url?: string;
	prerelease?: boolean;
}

function normalizeVersion(version: string): number[] {
	return version
		.replace(/^v/i, '')
		.split(/[-+]/)[0]
		.split('.')
		.map((part) => Number.parseInt(part, 10) || 0);
}

function compareVersions(left: string, right: string): number {
	const a = normalizeVersion(left);
	const b = normalizeVersion(right);
	const length = Math.max(a.length, b.length);
	for (let i = 0; i < length; i += 1) {
		const delta = (a[i] ?? 0) - (b[i] ?? 0);
		if (delta !== 0) return delta;
	}
	return 0;
}

function parseManifest(manifest: unknown): GitHubRelease[] {
	let releases: unknown[] = [];
	if (Array.isArray(manifest)) releases = manifest;
	if (manifest && typeof manifest === 'object') {
		const nestedReleases = (manifest as { releases?: unknown }).releases;
		if (Array.isArray(nestedReleases)) releases = nestedReleases;
	}
	return releases.flatMap((value): GitHubRelease[] => {
		if (!value || typeof value !== 'object') return [];
		const release = value as Record<string, unknown>;
		if (typeof release.tag_name !== 'string' || !RELEASE_VERSION.test(release.tag_name)) return [];
		return [
			{
				tag_name: release.tag_name,
				html_url: typeof release.html_url === 'string' ? release.html_url : undefined,
				prerelease: release.prerelease === true
			}
		];
	});
}

function releaseUrlFor(tagName: string, value: string | undefined): string {
	const fallback = `${RELEASES_PAGE_URL}/tag/${encodeURIComponent(tagName)}`;
	if (!value) return fallback;

	try {
		const url = new URL(value);
		const path = url.pathname.split('/').slice(1).map((part) => decodeURIComponent(part));
		if (
			url.protocol !== 'https:' ||
			url.hostname.toLowerCase() !== 'github.com' ||
			url.username !== '' ||
			url.password !== '' ||
			url.port !== '' ||
			path.length !== 5 ||
			path[0].toLowerCase() !== 'metaforismo' ||
			path[1].toLowerCase() !== 'sprintorio' ||
			path[2] !== 'releases' ||
			path[3] !== 'tag' ||
			path[4] !== tagName
		) {
			return fallback;
		}
		return url.href;
	} catch {
		return fallback;
	}
}

let version = $state(FALLBACK_VERSION);
let releaseUrl = $state('https://github.com/metaforismo/sprintorio');
let loaded = false;
let request: Promise<void> | null = null;

async function loadOnce(): Promise<boolean> {
	try {
		const response = await fetch(RELEASES_MANIFEST_URL, { cache: 'no-store' });
		if (!response.ok) return false;
		const releases = parseManifest(await response.json());
		const latest = releases
			.filter((release) => !release.prerelease && release.tag_name)
			.sort((a, b) => compareVersions(b.tag_name, a.tag_name))[0];
		if (!latest) return false;
		version = latest.tag_name;
		releaseUrl = releaseUrlFor(latest.tag_name, latest.html_url);
		return true;
	} catch {
		return false;
	}
}

async function loadWithRetry() {
	for (const delay of RELEASE_RETRY_DELAYS_MS) {
		if (delay > 0) await new Promise((resolve) => setTimeout(resolve, delay));
		if (await loadOnce()) {
			loaded = true;
			return;
		}
	}
}

/**
 * Reactive latest-release state. Consumers share one browser request at a
 * time; exhausted requests may be retried by a later component mount.
 */
export function useLatestRelease() {
	if (typeof window !== 'undefined' && !loaded && !request) {
		request = loadWithRetry().finally(() => {
			request = null;
		});
	}
	return {
		get version() {
			return version;
		},
		get releaseUrl() {
			return releaseUrl;
		}
	};
}
