import { ORIGIN } from '$lib/config/site';
export const prerender = true;
export function GET() {
 return new Response(`User-agent: *\nDisallow:\n\nSitemap: ${ORIGIN}/sitemap.xml\n`, { headers: { 'Content-Type': 'text/plain' } });
}
