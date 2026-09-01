// @ts-check
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLinksValidator from 'starlight-links-validator';
import { appTokensPlugin } from './src/lib/app-tokens.js';
import { installScriptsIntegration } from './src/lib/install.js';

// Canonical origin for the published site. Absolute URLs (canonical tags,
// the sitemap, install one-liners) derive from this value; do not hardcode a
// host elsewhere.
const site = 'https://beaverbacklog.com';

// https://astro.build/config
export default defineConfig({
	site,
	integrations: [
		installScriptsIntegration(),
		starlight({
			title: 'Beaver Backlog',
			description:
				'An issue tracker that lives in your repository, built for humans and coding agents working together.',
			plugins: [starlightLinksValidator()],
			sidebar: [
				{ label: 'Installation', slug: 'installation' },
				{ label: 'Quick start', slug: 'quick-start' },
				{ label: 'Command reference', slug: 'command-reference' },
				{ label: 'The issue file', slug: 'issue-file' },
				{ label: 'The web UI', slug: 'web-ui' },
				{ label: 'Working with coding agents', slug: 'coding-agents' },
				{ label: 'Configuration', slug: 'configuration' },
				{ label: 'Doctor', slug: 'doctor' },
			],
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/builtbystef/beaver-backlog',
				},
			],
			customCss: ['virtual:app-tokens.css', './src/styles/custom.css'],
			components: {
				Hero: './src/components/Hero.astro',
			},
		}),
	],
	vite: {
		plugins: [appTokensPlugin()],
		server: {
			fs: {
				allow: [
					fileURLToPath(new URL('.', import.meta.url)),
					fileURLToPath(new URL('..', import.meta.url)),
				],
			},
		},
	},
});
