import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';
import { unixInstallCommand, windowsInstallCommand } from '../src/lib/install.js';

const here = dirname(fileURLToPath(import.meta.url));
const siteDir = join(here, '..');
const dist = join(siteDir, 'dist');
const repo = join(siteDir, '..');

test('one-liners take their host from the origin they are given', () => {
	assert.equal(
		unixInstallCommand('https://example.test'),
		'curl -fsSL https://example.test/install.sh | sh',
	);
	assert.equal(
		windowsInstallCommand('https://example.test'),
		'irm https://example.test/install.ps1 | iex',
	);
});

function readBuilt(name) {
	const path = join(dist, name);
	try {
		return readFileSync(path);
	} catch (err) {
		if (err.code === 'ENOENT') {
			assert.fail(`${path} is missing; run npm run build first`);
		}
		throw err;
	}
}

test('build output copies the repository install scripts byte-for-byte', () => {
	for (const name of ['install.sh', 'install.ps1']) {
		const source = readFileSync(join(repo, name));
		assert.deepEqual(readBuilt(name), source, `${name} in dist should match the repository copy`);
	}
});

test('install scripts are not committed inside the site directory', () => {
	const listed = execFileSync('git', ['ls-files', '--', 'site'], {
		cwd: repo,
		encoding: 'utf8',
	});
	assert.doesNotMatch(listed, /(^|\n)site\/.*install\.(sh|ps1)(\n|$)/);
	assert.equal(existsSync(join(siteDir, 'public', 'install.sh')), false);
	assert.equal(existsSync(join(siteDir, 'public', 'install.ps1')), false);
});

function read(path) {
	try {
		return readFileSync(path, 'utf8');
	} catch (err) {
		if (err.code === 'ENOENT') {
			assert.fail(`${path} is missing; run npm run build first`);
		}
		throw err;
	}
}

function decode(html) {
	return html
		.replace(/<script[\s\S]*?<\/script>/gi, ' ')
		.replace(/<style[\s\S]*?<\/style>/gi, ' ')
		.replace(/<[^>]+>/g, '')
		.replace(/&amp;/g, '&')
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&#39;/g, "'")
		.replace(/&nbsp;/g, ' ');
}

function codeBlocks(html) {
	return [...html.matchAll(/<pre\b[^>]*>([\s\S]*?)<\/pre>/gi)].map((match) =>
		decode(match[1]).replace(/\s+/g, ' ').trim(),
	);
}

function configuredSite() {
	const source = read(join(siteDir, 'astro.config.mjs'));
	const match = source.match(/^const site = '([^']+)';$/m);
	assert.ok(match, 'canonical site URL should be declared in astro.config.mjs');
	return match[1];
}

test('Installation leads with the one-liners and keeps Go as alternatives', () => {
	const blocks = codeBlocks(read(join(dist, 'installation', 'index.html')));
	assert.ok(blocks.length >= 4, `expected at least four commands, got ${blocks.length}`);
	assert.equal(blocks[0], unixInstallCommand(configuredSite()));
	assert.equal(blocks[1], windowsInstallCommand(configuredSite()));
	const goAt = blocks.findIndex((block) => block.includes('go install'));
	const cloneAt = blocks.findIndex((block) => block.includes('git clone'));
	assert.ok(goAt > 1, 'go install should sit below the one-liners');
	assert.ok(cloneAt > 1, 'build-from-a-clone should sit below the one-liners');
});

test('published one-liners use the configured canonical URL', () => {
	const origin = configuredSite();
	const unix = unixInstallCommand(origin);
	const windows = windowsInstallCommand(origin);
	const installation = decode(read(join(dist, 'installation', 'index.html')));
	const landing = decode(read(join(dist, 'index.html')));
	assert.ok(installation.includes(unix), 'Installation should show the macOS/Linux one-liner');
	assert.ok(installation.includes(windows), 'Installation should show the Windows one-liner');
	assert.ok(landing.includes(unix), 'landing should show the macOS/Linux one-liner');
});
