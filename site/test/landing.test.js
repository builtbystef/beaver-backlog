import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';

const here = dirname(fileURLToPath(import.meta.url));
const dist = join(here, '..', 'dist');
const repo = join(here, '..', '..');

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
		.replace(/<[^>]+>/g, '')
		.replace(/&amp;/g, '&')
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&#39;/g, "'")
		.replace(/&nbsp;/g, ' ')
		.trim();
}

function firstCodeBlock(html) {
	const match = html.match(/<pre\b[^>]*>([\s\S]*?)<\/pre>/i);
	assert.ok(match, 'expected a <pre> code block');
	return decode(match[1]);
}

function builtCss() {
	const dir = join(dist, '_astro');
	const files = readdirSync(dir).filter((name) => name.endsWith('.css'));
	assert.ok(files.length > 0, 'expected compiled CSS in dist/_astro');
	return files.map((name) => read(join(dir, name))).join('\n');
}

function hexForms(value) {
	const match = value.match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i);
	if (!match) return [value];
	const hex = match[1].toLowerCase();
	if (hex.length === 3) {
		return [`#${hex}`, `#${hex[0]}${hex[0]}${hex[1]}${hex[1]}${hex[2]}${hex[2]}`];
	}
	const short =
		hex[0] === hex[1] && hex[2] === hex[3] && hex[4] === hex[5]
			? `#${hex[0]}${hex[2]}${hex[4]}`
			: null;
	return short ? [`#${hex}`, short] : [`#${hex}`];
}

function escapeRegExp(value) {
	return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function cssDeclares(css, name, value) {
	const forms = hexForms(value).map(escapeRegExp).join('|');
	return new RegExp(`--${name}:\\s*(?:${forms})\\b`, 'i').test(css);
}

function tokenBlock(css, selector) {
	const needle = selector + ' {';
	const start = css.indexOf(needle);
	assert.ok(start !== -1, `expected ${selector} in ${css.slice(0, 80)}`);
	const brace = start + needle.length - 1;
	let depth = 0;
	for (let i = brace; i < css.length; i++) {
		if (css[i] === '{') depth++;
		else if (css[i] === '}') {
			depth--;
			if (depth === 0) return css.slice(brace + 1, i);
		}
	}
	assert.fail(`unclosed ${selector}`);
}

function tokenValues(block) {
	const values = {};
	for (const match of block.matchAll(/--([a-z0-9-]+):\s*([^;]+);/g)) {
		values[match[1]] = match[2].trim();
	}
	return values;
}

const landing = read(join(dist, 'index.html'));
const installation = read(join(dist, 'installation', 'index.html'));

test('landing renders the logo, pitch, install command, features, docs link, and GitHub link', () => {
	assert.match(landing, /<img\b[^>]*alt="Beaver Backlog"/i);

	assert.match(
		landing,
		/An issue tracker that lives in your repository, built for humans and coding agents working together/,
	);

	const command = firstCodeBlock(installation);
	assert.ok(command.length > 0, 'Installation should lead with a command');
	assert.ok(
		landing.includes(command) || landing.includes(command.replace(/</g, '&lt;')),
		`landing should show the Installation command ${JSON.stringify(command)}`,
	);

	for (const feature of [
		'Markdown-first',
		'Local by default',
		'Version-control-friendly',
		'Agent-friendly',
		'Nothing hidden',
	]) {
		assert.ok(landing.includes(feature), `features row should include ${feature}`);
	}

	assert.match(landing, /href="\/installation\/"/);
	assert.match(landing, /href="https:\/\/github\.com\/builtbystef\/beaver-backlog"/);
});

test('landing install command is the one Installation leads with', () => {
	const fromDocs = firstCodeBlock(installation);
	const fromLanding = firstCodeBlock(landing);
	assert.equal(fromLanding, fromDocs);
});

function screenshotSlot(html) {
	const match = html.match(/<section[^>]*data-screenshot-slot[\s\S]*?<\/section>/);
	assert.ok(match, 'expected data-screenshot-slot on the landing page');
	return match[0];
}

test('landing shows two to five screenshots, the board first, paired by theme', () => {
	const slot = screenshotSlot(landing);
	const imgs = [...slot.matchAll(/<img\b[^>]*>/gi)].map((m) => m[0]);
	assert.ok(imgs.length >= 4 && imgs.length <= 10, `landing should show 2–5 views as light/dark pairs, got ${imgs.length} images`);

	const alts = imgs.map((tag) => {
		const m = tag.match(/\balt="([^"]+)"/i);
		assert.ok(m && m[1].trim(), `screenshot missing alternative text: ${tag}`);
		return m[1];
	});
	assert.match(alts[0], /board/i);

	const light = imgs.filter((tag) => /dark:sl-hidden/.test(tag));
	const dark = imgs.filter((tag) => /light:sl-hidden/.test(tag));
	assert.equal(light.length, dark.length, 'each landing screenshot needs a light and a dark image');
	assert.ok(light.length >= 2 && light.length <= 5, `expected 2–5 views, got ${light.length}`);
	assert.equal(light.length + dark.length, imgs.length, 'every landing screenshot must hide in the other theme');
});

test('neutral scale and accent take their values from the application tokens', () => {
	const app = read(join(repo, 'internal', 'web', 'styles', 'tailwind.css'));
	const light = tokenValues(tokenBlock(app, ':root'));
	const dark = tokenValues(tokenBlock(app, ':root[data-theme="dark"]'));
	const css = builtCss();

	for (const name of ['canvas', 'surface', 'ink', 'ink-muted', 'accent']) {
		assert.ok(light[name], `application light token --${name} missing`);
		assert.ok(dark[name], `application dark token --${name} missing`);
		assert.ok(
			cssDeclares(css, name, light[name]),
			`site CSS should carry light --${name}: ${light[name]}`,
		);
		assert.ok(
			cssDeclares(css, name, dark[name]),
			`site CSS should carry dark --${name}: ${dark[name]}`,
		);
	}
});

test('both palettes exist so light and dark follow the reader', () => {
	assert.match(landing, /prefers-color-scheme/);
	assert.match(landing, /data-theme/);
	const css = builtCss();
	assert.match(css, /\[data-theme=['"]?light['"]?\]/);
	assert.match(css, /\[data-theme=['"]?dark['"]?\]/);
});

test('layout does not force the page body to scroll sideways', () => {
	const styles = builtCss() + landing;
	assert.match(styles, /overflow-x:\s*clip/);
	assert.match(styles, /minmax\(min\(100%/);
});

function canonicalSite() {
	const source = read(join(here, '..', 'astro.config.mjs'));
	const match = source.match(/^const site = '([^']+)';$/m);
	assert.ok(match, 'canonical site URL should be declared in astro.config.mjs');
	return match[1];
}

function pngSize(buffer) {
	assert.equal(buffer.toString('latin1', 1, 4), 'PNG', 'og.png should be a PNG');
	return { width: buffer.readUInt32BE(16), height: buffer.readUInt32BE(20) };
}

test('every page carries the Open Graph card at the canonical URL', () => {
	const url = `${canonicalSite()}/og.png`;
	for (const html of [landing, installation]) {
		assert.ok(html.includes(`<meta property="og:image" content="${url}"`), 'expected og:image');
		assert.ok(html.includes(`<meta name="twitter:image" content="${url}"`), 'expected twitter:image');
	}
	const { width, height } = pngSize(readFileSync(join(dist, 'og.png')));
	assert.deepEqual({ width, height }, { width: 1200, height: 630 });
});

test('robots.txt allows every crawler and names the sitemap at the canonical URL', () => {
	const robots = read(join(dist, 'robots.txt'));
	assert.match(robots, /^User-agent: \*\nAllow: \/\n/);
	assert.ok(robots.includes(`Sitemap: ${canonicalSite()}/sitemap-index.xml`), robots);
	assert.ok(existsSync(join(dist, 'sitemap-index.xml')), 'the sitemap the robots file names should exist');
});
