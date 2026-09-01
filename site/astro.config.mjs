// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLinksValidator from 'starlight-links-validator';

// Canonical origin for the published site. Absolute URLs (canonical tags,
// the sitemap, later install one-liners) derive from this value; do not
// hardcode a host elsewhere.
const site = 'https://beaverbacklog.com';

// https://astro.build/config
export default defineConfig({
	site,
	integrations: [
		starlight({
			title: 'Beaver Backlog',
			description:
				'An issue tracker that lives in your repository, built for humans and coding agents working together.',
			plugins: [starlightLinksValidator()],
			sidebar: [{ label: 'Installation', slug: 'installation' }],
		}),
	],
});
