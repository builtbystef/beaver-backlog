import type { APIRoute } from 'astro';

// robots.txt names the sitemap by its absolute URL, so it is generated from the
// configured site rather than committed with a host written into it.
export const GET: APIRoute = ({ site }) => {
	const sitemap = new URL('/sitemap-index.xml', site).href;
	const body = ['User-agent: *', 'Allow: /', '', `Sitemap: ${sitemap}`, ''].join('\n');
	return new Response(body, { headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
};
