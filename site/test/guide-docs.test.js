import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';

const here = dirname(fileURLToPath(import.meta.url));
const dist = join(here, '..', 'dist');

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

function page(slug) {
	return read(join(dist, slug, 'index.html'));
}

function decode(html) {
	return html
		.replace(/<script[\s\S]*?<\/script>/gi, ' ')
		.replace(/<style[\s\S]*?<\/style>/gi, ' ')
		.replace(/<[^>]+>/g, ' ')
		.replace(/&amp;/g, '&')
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&#39;/g, "'")
		.replace(/&nbsp;/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

function sidebar(html) {
	const match = html.match(/<ul class="top-level[\s\S]*?<\/ul>/);
	assert.ok(match, 'expected a top-level sidebar list');
	return decode(match[0]);
}

function allGuideHtml() {
	return ['web-ui', 'coding-agents', 'configuration', 'doctor'].map(page).join('\n');
}

test('sidebar lists The web UI, Working with coding agents, Configuration, and Doctor', () => {
	const nav = sidebar(page('installation'));
	for (const label of ['The web UI', 'Working with coding agents', 'Configuration', 'Doctor']) {
		assert.ok(nav.includes(label), `sidebar should list ${label}`);
	}

	const installation = page('installation');
	for (const href of ['/web-ui/', '/coding-agents/', '/configuration/', '/doctor/']) {
		assert.ok(installation.includes(`href="${href}"`), `sidebar should link to ${href}`);
	}

	assert.match(page('web-ui'), /<h1[^>]*>The web UI<\/h1>/);
	assert.match(page('coding-agents'), /<h1[^>]*>Working with coding agents<\/h1>/);
	assert.match(page('configuration'), /<h1[^>]*>Configuration<\/h1>/);
	assert.match(page('doctor'), /<h1[^>]*>Doctor<\/h1>/);
});

test('web UI page describes the views and documents port behavior', () => {
	const text = decode(page('web-ui'));

	for (const view of ['board', 'list', 'graph', 'issue', 'doctor']) {
		assert.match(text, new RegExp(view, 'i'), `web UI page should describe the ${view} view`);
	}

	assert.ok(text.includes('2328'), 'web UI page should document the default port 2328');
	assert.match(text, /scan/i);
	assert.match(text, /taken|occupied|in use/i);
	assert.ok(text.includes('--port'), 'web UI page should document --port');
	assert.ok(text.includes('beaver serve'), 'web UI page should cover beaver serve');
	assert.ok(text.includes('--as'), 'web UI page should document attribution with --as');
});

test('coding agents page documents actor resolution and setting BEAVER_BACKLOG_ACTOR', () => {
	const html = page('coding-agents');
	const text = decode(html);

	assert.ok(text.includes('--as'), 'coding agents page should name --as');
	assert.ok(text.includes('BEAVER_BACKLOG_ACTOR'), 'coding agents page should name BEAVER_BACKLOG_ACTOR');
	assert.match(text, /per-machine|user config/i);

	const asAt = text.indexOf('--as');
	const envAt = text.indexOf('BEAVER_BACKLOG_ACTOR');
	const configAt = text.search(/per-machine|user config/i);
	assert.ok(asAt !== -1 && envAt !== -1 && configAt !== -1);
	assert.ok(asAt < envAt, 'resolution order should name --as before BEAVER_BACKLOG_ACTOR');
	assert.ok(envAt < configAt, 'resolution order should name BEAVER_BACKLOG_ACTOR before per-machine user config');

	assert.match(html, /BEAVER_BACKLOG_ACTOR=/);
});

test('coding agents page shows --body-file - and beaver list --ready', () => {
	const html = page('coding-agents');
	const text = decode(html);

	assert.match(html, /--body-file\s+-/);
	assert.match(html, /beaver create/);
	assert.ok(text.includes('beaver list --ready') || html.includes('list --ready'));
	assert.match(text, /ready/i);
});

test('coding agents page states that a claim is advisory and each agent needs its own working tree', () => {
	const text = decode(page('coding-agents'));

	assert.match(text, /claim/i);
	assert.match(text, /advisory/i);
	assert.match(text, /not a lock|not.+lock/i);
	assert.match(text, /working tree/i);
	assert.match(text, /unsupported/i);
});

test('configuration page states committed config, per-machine identity, and no VCS', () => {
	const text = decode(page('configuration'));

	assert.ok(text.includes('.beaver/config.yml'));
	assert.match(text, /committed/i);
	assert.match(text, /shared/i);
	assert.match(text, /identity/i);
	assert.match(text, /per-machine/i);
	assert.match(text, /never in the repository|never committed|not in the repository/i);
	assert.match(text, /version-control|version control|VCS/i);
	assert.match(text, /never runs|does not run|never invokes/i);
});

test('doctor page states skip-and-warn and that --fix never removes data', () => {
	const text = decode(page('doctor'));

	assert.match(text, /invalid/i);
	assert.match(text, /skip/i);
	assert.match(text, /warning/i);
	assert.match(text, /crash/i);
	assert.ok(text.includes('--fix'));
	assert.match(text, /unambiguous|safe to repair|mechanically safe/i);
	assert.match(text, /never removes data/i);
});

test('contributor material and ADRs are linked, not republished', () => {
	const html = allGuideHtml();
	const text = decode(html);

	assert.match(html, /CONTRIBUTING\.md/);
	assert.match(html, /docs\/adr/);

	assert.ok(
		!text.includes('Thanks for your interest in contributing!'),
		'pages must not copy CONTRIBUTING.md',
	);
	assert.ok(
		!text.includes('Two kinds of file problem are deliberately distinguished'),
		'pages must not copy ADR 0003',
	);
	assert.ok(
		!text.includes('Beaver Backlog serves solo devs, teams, open-source contributors'),
		'pages must not copy ADR 0004',
	);
});
