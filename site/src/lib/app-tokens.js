// Reads the application token table and the Installation page so the site
// cannot pick a second palette or a second install command that would drift.
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const sourcePath = join(here, '../../../internal/web/styles/tailwind.css');
const installationPath = join(here, '../content/docs/installation.md');

export function leadingInstallCommand() {
	const source = readFileSync(installationPath, 'utf8');
	const match = source.match(/```[^\n]*\n([\s\S]*?)```/);
	if (!match) {
		throw new Error('Installation page has no command for the landing page to show');
	}
	return match[1].trim();
}

function extractBlock(css, selector) {
	const needle = selector + ' {';
	const start = css.indexOf(needle);
	if (start === -1) {
		throw new Error(`application tokens missing ${selector}`);
	}
	const brace = start + needle.length - 1;
	let depth = 0;
	for (let i = brace; i < css.length; i++) {
		if (css[i] === '{') depth++;
		else if (css[i] === '}') {
			depth--;
			if (depth === 0) return css.slice(brace + 1, i);
		}
	}
	throw new Error(`unclosed ${selector} in application tokens`);
}

function parseVars(block) {
	const values = {};
	for (const match of block.matchAll(/--([a-z0-9-]+):\s*([^;]+);/g)) {
		values[match[1]] = match[2].trim();
	}
	return values;
}

function decls(vars) {
	return Object.entries(vars)
		.map(([name, value]) => `\t--${name}: ${value};`)
		.join('\n');
}

const starlightMap = `
	--sl-color-bg: var(--canvas);
	--sl-color-bg-nav: var(--surface);
	--sl-color-bg-sidebar: var(--surface);
	--sl-color-bg-inline-code: var(--surface-hover);
	--sl-color-text: var(--ink-muted);
	--sl-color-white: var(--ink);
	--sl-color-gray-1: var(--ink);
	--sl-color-gray-2: var(--ink-muted);
	--sl-color-gray-3: var(--ink-subtle);
	--sl-color-gray-4: var(--line-strong);
	--sl-color-gray-5: var(--line);
	--sl-color-gray-6: var(--surface);
	--sl-color-gray-7: var(--surface-raised);
	--sl-color-black: var(--canvas);
	--sl-color-accent: var(--accent);
	--sl-color-accent-high: var(--accent-hover);
	--sl-color-accent-low: var(--accent-soft);
	--sl-color-text-accent: var(--accent);
	--sl-color-bg-accent: var(--accent);
	--sl-color-text-invert: var(--ink-on-accent);
	--sl-color-hairline: var(--line);
	--sl-color-hairline-light: var(--line);
	--sl-color-hairline-shade: var(--line-strong);
`;

export function generateAppTokensCss() {
	const css = readFileSync(sourcePath, 'utf8');
	const light = parseVars(extractBlock(css, ':root'));
	const dark = parseVars(extractBlock(css, ':root[data-theme="dark"]'));
	for (const name of ['canvas', 'surface', 'ink', 'accent']) {
		if (!light[name] || !dark[name]) {
			throw new Error(`application tokens missing --${name}`);
		}
	}
	return `/* Values taken from the application token table. Do not edit by hand. */
:root[data-theme='light'] {
${decls(light)}
${starlightMap}
}
:root[data-theme='dark'] {
${decls(dark)}
${starlightMap}
}
`;
}

export function appTokensPlugin() {
	return {
		name: 'app-tokens',
		resolveId(id) {
			if (id === 'virtual:app-tokens.css') return '\0virtual:app-tokens.css';
			if (id === 'virtual:install-command') return '\0virtual:install-command';
		},
		load(id) {
			if (id === '\0virtual:app-tokens.css') {
				this.addWatchFile(sourcePath);
				return generateAppTokensCss();
			}
			if (id === '\0virtual:install-command') {
				this.addWatchFile(installationPath);
				return `export const installCommand = ${JSON.stringify(leadingInstallCommand())};`;
			}
		},
	};
}
