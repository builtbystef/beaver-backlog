import assert from 'node:assert/strict';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { basename, dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';

const here = dirname(fileURLToPath(import.meta.url));
const repo = join(here, '..', '..');
const shots = join(repo, 'docs', 'assets', 'screenshots');

const views = ['board', 'list', 'graph', 'issue'];
const palettes = ['light', 'dark'];
const files = views.flatMap((view) => palettes.map((palette) => `${view}-${palette}.png`));

// 300 KiB keeps a landing page that shows three pairs reasonable to load even
// if the browser fetches both palettes.
const maxBytes = 300 * 1024;

const skipDirs = new Set(['.git', 'node_modules', 'dist', '.implement-loop', '.astro']);

function walk(dir, acc = []) {
	for (const entry of readdirSync(dir, { withFileTypes: true })) {
		if (skipDirs.has(entry.name)) continue;
		const path = join(dir, entry.name);
		if (entry.isDirectory()) walk(path, acc);
		else acc.push(path);
	}
	return acc;
}

function readmeRefs(md) {
	const refs = [];
	for (const match of md.matchAll(/!\[[^\]]*]\(([^)]+)\)/g)) {
		refs.push(match[1].trim());
	}
	for (const match of md.matchAll(/\b(?:src|srcset)="([^"]+)"/gi)) {
		refs.push(match[1].trim());
	}
	return refs.filter((ref) => ref && !/^(https?:|#|data:)/i.test(ref));
}

test('the screenshot set covers board, list, graph, and issue in both palettes', () => {
	for (const name of files) {
		const path = join(shots, name);
		assert.ok(existsSync(path), `missing ${relative(repo, path)}`);
	}
});

test('each screenshot exists exactly once in the repository', () => {
	const wanted = new Set(files);
	const found = new Map();
	for (const path of walk(repo)) {
		const name = basename(path);
		if (!wanted.has(name)) continue;
		if (!found.has(name)) found.set(name, []);
		found.get(name).push(relative(repo, path));
	}
	for (const name of files) {
		const copies = found.get(name) || [];
		assert.equal(
			copies.length,
			1,
			`${name} should exist once, found ${copies.length}: ${copies.join(', ')}`,
		);
		assert.equal(copies[0], join('docs', 'assets', 'screenshots', name));
	}
});

test('screenshots are sized for the web', () => {
	for (const name of files) {
		const bytes = statSync(join(shots, name)).size;
		assert.ok(bytes > 0, `${name} is empty`);
		assert.ok(bytes <= maxBytes, `${name} is ${bytes} bytes, want <= ${maxBytes}`);
	}
});

test('every README image reference resolves to a file that exists', () => {
	const readme = readFileSync(join(repo, 'README.md'), 'utf8');
	const refs = readmeRefs(readme);
	assert.ok(refs.length > 0, 'README should reference images');
	for (const ref of refs) {
		const path = join(repo, ref);
		assert.ok(existsSync(path), `README image ${ref} does not exist`);
	}
	for (const name of files) {
		assert.ok(
			refs.some((ref) => ref.endsWith(`screenshots/${name}`) || ref.endsWith(`screenshots/${name}`.replaceAll('\\', '/'))),
			`README should reference ${name}`,
		);
	}
});
