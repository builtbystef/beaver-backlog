import assert from 'node:assert/strict';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { test } from 'node:test';
import { basename, dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const repo = join(here, '..', '..');
const dist = join(here, '..', 'dist');
const shots = join(repo, 'docs', 'assets', 'screenshots');

const views = ['board', 'list', 'graph', 'issue'];
const palettes = ['light', 'dark'];
const files = views.flatMap((view) => palettes.map((palette) => `${view}-${palette}.png`));

// The committed captures are 2x (2880 by 1800) so a high-density screen gets
// a sharp picture; the README shows them as they are. The site never serves
// them directly: Astro derives the WebP files the landing page loads, and a
// second test below holds those to the budget that keeps a page showing three
// pairs reasonable to load even if the browser fetches both palettes.
const maxBytes = 512 * 1024;
const maxServedBytes = 300 * 1024;
const captureWidth = 2880;
const captureHeight = 1800;

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

function pngSize(path) {
	const header = readFileSync(path).subarray(0, 24);
	assert.equal(header.toString('latin1', 1, 4), 'PNG', `${relative(repo, path)} is not a PNG`);
	return { width: header.readUInt32BE(16), height: header.readUInt32BE(20) };
}

test('screenshots are 2x captures within the source budget', () => {
	for (const name of files) {
		const path = join(shots, name);
		const bytes = statSync(path).size;
		assert.ok(bytes > 0, `${name} is empty`);
		assert.ok(bytes <= maxBytes, `${name} is ${bytes} bytes, want <= ${maxBytes}`);
		assert.deepEqual(
			pngSize(path),
			{ width: captureWidth, height: captureHeight },
			`${name} should be a ${captureWidth}x${captureHeight} capture`,
		);
	}
});

test('the landing serves derived WebP files with a density-aware source set', () => {
	const landing = readFileSync(join(dist, 'index.html'), 'utf8');
	const slot = landing.match(/<section[^>]*data-screenshot-slot[\s\S]*?<\/section>/);
	assert.ok(slot, 'expected data-screenshot-slot on the landing page');
	const imgs = [...slot[0].matchAll(/<img\b[^>]*>/gi)].map((m) => m[0]);
	assert.ok(imgs.length > 0, 'expected screenshots on the landing page');

	const served = new Set();
	for (const tag of imgs) {
		const srcset = tag.match(/\bsrcset="([^"]+)"/i);
		assert.ok(srcset, `screenshot should carry a srcset: ${tag}`);
		const candidates = srcset[1].split(',').map((c) => c.trim().split(/\s+/));
		assert.ok(candidates.length >= 2, `expected a 1x and a 2x candidate: ${srcset[1]}`);
		for (const [url] of candidates) {
			assert.match(url, /\.webp$/, `derived screenshot should be WebP: ${url}`);
			served.add(url);
		}
		assert.match(tag, /\bsizes="/i, `screenshot with a srcset needs sizes: ${tag}`);
		assert.doesNotMatch(tag, /\.png"/i, `landing should not serve the PNG source: ${tag}`);
	}

	for (const url of served) {
		const path = join(dist, url);
		assert.ok(existsSync(path), `${url} is referenced but not built`);
		const bytes = statSync(path).size;
		assert.ok(bytes <= maxServedBytes, `${url} is ${bytes} bytes, want <= ${maxServedBytes}`);
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
