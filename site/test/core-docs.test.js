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

test('sidebar lists Quick start, Command reference, and The issue file', () => {
	const nav = sidebar(page('installation'));
	for (const label of ['Installation', 'Quick start', 'Command reference', 'The issue file']) {
		assert.ok(nav.includes(label), `sidebar should list ${label}`);
	}

	const installation = page('installation');
	for (const href of ['/quick-start/', '/command-reference/', '/issue-file/']) {
		assert.ok(installation.includes(`href="${href}"`), `sidebar should link to ${href}`);
	}

	assert.match(page('quick-start'), /<h1[^>]*>Quick start<\/h1>/);
	assert.match(page('command-reference'), /<h1[^>]*>Command reference<\/h1>/);
	assert.match(page('issue-file'), /<h1[^>]*>The issue file<\/h1>/);
});

const commands = [
	'init',
	'create',
	'list',
	'show',
	'start',
	'done',
	'cancel',
	'reopen',
	'update',
	'note',
	'delete',
	'doctor',
	'serve',
	'whoami',
	'version',
];

test('command reference documents every command and its flags', () => {
	const html = page('command-reference');
	const text = decode(html);

	for (const cmd of commands) {
		assert.ok(
			text.includes(`beaver ${cmd}`),
			`command reference should document beaver ${cmd}`,
		);
	}

	const flags = [
		['create', ['--body', '--body-file', '--label', '--priority', '--depends-on', '--parent']],
		['list', ['--state', '--ready', '--blocked', '--label', '--priority', '--assignee', '--parent', '--search']],
		['start', ['--as', '--force']],
		[
			'update',
			[
				'--title',
				'--body',
				'--body-file',
				'--assignee',
				'--unassign',
				'--priority',
				'--label',
				'--depends-on',
				'--parent',
				'--no-parent',
			],
		],
		['note', ['--as']],
		['doctor', ['--fix']],
		['serve', ['--port', '--as']],
		['whoami', ['--as']],
	];
	for (const [, names] of flags) {
		for (const flag of names) {
			assert.ok(text.includes(flag), `command reference should document ${flag}`);
		}
	}

	assert.ok(text.includes('beaver list --ready') || text.includes('list --ready'));
	assert.match(text, /todo/);
	assert.match(text, /every dependency/);
	assert.match(text, /\bdone\b/);
});

test('command reference states exit codes and format auto-detection', () => {
	const text = decode(page('command-reference'));

	assert.match(text, /\b0\b[^.]{0,40}success/i);
	assert.match(text, /\b1\b[^.]{0,60}runtime/i);
	assert.match(text, /\b2\b[^.]{0,40}usage/i);
	assert.match(text, /\b3\b[^.]{0,50}not found/i);

	assert.ok(text.includes('--format'));
	assert.match(text, /\bhuman\b/);
	assert.match(text, /\bjson\b/i);
	assert.match(text, /auto-detect/i);
});

test('issue file page shows a complete example and every frontmatter field', () => {
	const html = page('issue-file');
	const text = decode(html);

	assert.ok(html.includes('---'), 'example file should include YAML fences');
	assert.match(html, /## Notes/);

	for (const field of [
		'id',
		'title',
		'state',
		'assignee',
		'priority',
		'labels',
		'depends_on',
		'parent',
		'created',
		'updated',
	]) {
		assert.ok(text.includes(field), `issue file page should document ${field}`);
	}

	for (const state of ['todo', 'in-progress', 'done', 'cancelled']) {
		assert.ok(text.includes(state), `issue file page should name state ${state}`);
	}
});

test('issue file page states the three rules for a safe hand edit', () => {
	const text = decode(page('issue-file'));

	assert.match(text, /notes/i);
	assert.match(text, /leave/i);
	assert.match(text, /never change the `?id`?/i);
	assert.match(text, /follow up/i);
	assert.match(text, /note/i);
});

test('quick start walks the first session with real console output', () => {
	const html = page('quick-start');
	const text = decode(html);

	for (const cmd of ['init', 'create', 'list', 'start', 'note', 'update', 'done']) {
		assert.ok(text.includes(`beaver ${cmd}`), `quick start should run beaver ${cmd}`);
	}

	assert.match(html, /Initialized empty Beaver Backlog store/);
	assert.match(html, /Created /);
	assert.match(html, /Started /);
	assert.match(html, /claimed for/);
	assert.match(html, /Added note to/);
	assert.match(html, /Updated /);
	assert.match(html, /Marked .+ done/);
});

test('quick start states that state changes are verbs and other fields go through update', () => {
	const text = decode(page('quick-start'));
	assert.match(text, /state/i);
	assert.match(text, /update/);
	assert.match(text, /verb/i);
});
