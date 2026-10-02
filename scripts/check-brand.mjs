import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const read = (path) => readFileSync(resolve(root, path), 'utf8');
const repository = 'metaforismo/sprintorio';
const ui = JSON.parse(read('UI/package.json'));
const lock = JSON.parse(read('UI/package-lock.json'));
const site = JSON.parse(read('WEB/package.json'));
const siteLock = JSON.parse(read('WEB/package-lock.json'));
assert.equal(ui.name, 'sprintorio-ui');
assert.equal(lock.name, ui.name);
assert.equal(lock.version, ui.version);
assert.equal(lock.packages[''].version, ui.version);
assert.equal(site.name, 'sprintorio-site');
assert.equal(siteLock.name, site.name);
assert.equal(siteLock.version, site.version);
assert.equal(siteLock.packages[''].version, site.version);
assert.match(read('BE/go.mod'), /^module github\.com\/metaforismo\/sprintorio\/BE\n/);
assert.match(read('README.md'), /# Sprintorio\n/);
assert.ok(read('README.md').includes(`https://github.com/${repository}.git`));
assert.ok(read('UI/src/lib/release.ts').includes(`'${repository}'`));
assert.ok(read('WEB/src/lib/release.svelte.ts').includes(`https://github.com/${repository}/releases`));
assert.ok(read('.github/workflows/build.yml').includes('ghcr.io/${{ github.repository_owner }}'));
for (const path of [
  'assets/logo.svg', 'assets/logo_primary.svg', 'assets/logo_black.svg', 'assets/logo_white.svg',
  'assets/favicon_logo.svg', 'UI/static/favicon.svg', 'UI/src/lib/assets/favicon.svg',
  'WEB/static/logo_primary.svg', 'WEB/static/logo_white.svg', 'WEB/static/favicon.svg',
  'WEB/static/favicon_logo.svg', 'WEB/src/lib/assets/favicon.svg'
]) {
  assert.match(read(path), /aria-label="Sprintorio"/);
}
for (const release of JSON.parse(read('UI/static/releases.json'))) {
  assert.ok(release.html_url.startsWith(`https://github.com/${repository}/releases/`));
}
assert.match(read('LICENSE'), /Apache License/);
assert.match(read('NOTICE'), /Copyright/);
console.log('PASS: Sprintorio identity, module, repository links, legal notices, and assets.');
