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

// The type stack. Both faces are self-hosted from npm so a page never waits on
// a third-party font host, and both are variable fonts, so one file each.
const fontSans =
	"'Inter Variable', ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif";
const fontMono =
	"'JetBrains Mono Variable', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";

// https://astro.build/config
export default defineConfig({
	site,
	integrations: [
		installScriptsIntegration(),
		starlight({
			title: 'Beaver Backlog',
			description:
				'An issue tracker that lives in your repository, built for humans and coding agents working together.',
			logo: {
				light: '../docs/assets/logo-icon.svg',
				dark: '../docs/assets/logo-icon-dark.svg',
				alt: 'Beaver Backlog',
			},
			// The social preview card. Its source and renderer live in og/.
			head: [
				{ tag: 'meta', attrs: { property: 'og:image', content: new URL('/og.png', site).href } },
				{ tag: 'meta', attrs: { property: 'og:image:width', content: '1200' } },
				{ tag: 'meta', attrs: { property: 'og:image:height', content: '630' } },
				{ tag: 'meta', attrs: { name: 'twitter:image', content: new URL('/og.png', site).href } },
			],
			pagefind: false,
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
			customCss: [
				'@fontsource-variable/inter',
				'@fontsource-variable/jetbrains-mono',
				'virtual:app-tokens.css',
				'./src/styles/custom.css',
				'./src/styles/landing.css',
			],
			components: {
				Header: './src/components/Header.astro',
				Footer: './src/components/Footer.astro',
				Hero: './src/components/Hero.astro',
			},
			// Code blocks draw on the application palette like everything else:
			// an ink border with a hard offset shadow, the way the landing draws
			// every card, and the site's monospace face. The syntax themes are
			// GitHub's, which most readers already know.
			expressiveCode: {
				themes: ['github-dark-default', 'github-light'],
				styleOverrides: {
					borderRadius: '0.75rem',
					borderColor: 'var(--ink)',
					borderWidth: '0px',
					codeBackground: 'var(--surface)',
					codeFontFamily: fontMono,
					codeFontSize: '0.8125rem',
					codeLineHeight: '1.7',
					codePaddingBlock: '0.875rem',
					codePaddingInline: '1.125rem',
					uiFontFamily: fontSans,
					uiFontSize: '0.75rem',
					frames: {
						shadowColor: 'transparent',
						frameBoxShadowCssValue: '3px 3px 0 var(--ink)',
						editorBackground: 'var(--surface)',
						editorTabBarBackground: 'var(--surface-hover)',
						editorTabBarBorderBottomColor: 'var(--line)',
						editorActiveTabBackground: 'var(--surface)',
						editorActiveTabForeground: 'var(--ink)',
						editorActiveTabIndicatorTopColor: 'transparent',
						editorActiveTabIndicatorBottomColor: 'transparent',
						editorActiveTabIndicatorHeight: '0px',
						editorTabBorderRadius: '0',
						terminalBackground: 'var(--surface)',
						terminalTitlebarBackground: 'var(--surface-hover)',
						terminalTitlebarBorderBottomColor: 'var(--line)',
						terminalTitlebarForeground: 'var(--ink-subtle)',
						terminalTitlebarDotsForeground: 'var(--ink-subtle)',
						terminalTitlebarDotsOpacity: '0.4',
						inlineButtonBackground: 'var(--surface-hover)',
						inlineButtonBorder: 'var(--line-strong)',
						inlineButtonForeground: 'var(--ink-muted)',
						tooltipSuccessBackground: 'var(--ink)',
						tooltipSuccessForeground: 'var(--canvas)',
					},
				},
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
