import assert from 'node:assert/strict';
import test from 'node:test';
import {
	githubRepositoryHomeUrl,
	safeGitHubAppInstallUrl,
	safeGitHubBranchUrl,
	safeGitHubCommitUrl,
	safeGitHubPullRequestUrl,
	safeGitHubReleaseUrl,
	safeGitHubRepositoryUrl
} from '../../src/lib/security/github-url.ts';

const repository = 'metaforismo/sprintorio';

test('accepts only HTTPS GitHub URLs under the expected repository', () => {
	assert.equal(
		safeGitHubRepositoryUrl('https://github.com/metaforismo/sprintorio/compare/v1...v2', repository),
		'https://github.com/metaforismo/sprintorio/compare/v1...v2'
	);
	assert.equal(safeGitHubRepositoryUrl('http://github.com/metaforismo/sprintorio', repository), null);
	assert.equal(safeGitHubRepositoryUrl('https://github.com.example.com/metaforismo/sprintorio', repository), null);
	assert.equal(safeGitHubRepositoryUrl('https://github.com/attacker/sprintorio', repository), null);
	assert.equal(
		safeGitHubRepositoryUrl('https://github.com@attacker.example/metaforismo/sprintorio', repository),
		null
	);
	assert.equal(githubRepositoryHomeUrl('metaforismo/sprintorio'), 'https://github.com/metaforismo/sprintorio');
	assert.equal(githubRepositoryHomeUrl('metaforismo/sprintorio/extra'), null);
});

test('validates GitHub resource paths', () => {
	assert.equal(
		safeGitHubPullRequestUrl('https://github.com/metaforismo/sprintorio/pull/46', repository),
		'https://github.com/metaforismo/sprintorio/pull/46'
	);
	assert.equal(safeGitHubPullRequestUrl('https://github.com/metaforismo/sprintorio/issues/46', repository), null);
	assert.equal(
		safeGitHubBranchUrl('https://github.com/metaforismo/sprintorio/tree/feature/link-policy', repository),
		'https://github.com/metaforismo/sprintorio/tree/feature/link-policy'
	);
	assert.equal(safeGitHubBranchUrl('https://github.com/other-owner/other/tree/main', repository), null);
	assert.equal(
		safeGitHubCommitUrl('https://github.com/metaforismo/sprintorio/commit/0123456789abcdef', repository),
		'https://github.com/metaforismo/sprintorio/commit/0123456789abcdef'
	);
	assert.equal(safeGitHubCommitUrl('https://github.com/metaforismo/sprintorio/commit/not-a-sha', repository), null);
	assert.equal(
		safeGitHubReleaseUrl('https://github.com/metaforismo/sprintorio/releases/tag/v0.1.12', repository, 'v0.1.12'),
		'https://github.com/metaforismo/sprintorio/releases/tag/v0.1.12'
	);
	assert.equal(
		safeGitHubReleaseUrl('https://github.com/metaforismo/sprintorio/releases/tag/v0.1.11', repository, 'v0.1.12'),
		null
	);
});

test('validates GitHub App installation destinations', () => {
	assert.equal(
		safeGitHubAppInstallUrl('https://github.com/apps/sprintorio/installations/new?state=workspace-id'),
		'https://github.com/apps/sprintorio/installations/new?state=workspace-id'
	);
	assert.equal(safeGitHubAppInstallUrl('https://attacker.example/apps/sprintorio/installations/new'), null);
	assert.equal(safeGitHubAppInstallUrl('https://github.com/marketplace/sprintorio'), null);
});
