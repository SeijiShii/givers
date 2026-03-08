import { defineMiddleware } from 'astro:middleware';

const SKIP_PREFIXES = ['/_astro/', '/api/', '/uploads/', '/favicon', '/robots.txt', '/sitemap'];

function parseAcceptLanguage(header: string): string[] {
  return header
    .split(',')
    .map((part) => {
      const [lang, q] = part.trim().split(';q=');
      return { lang: lang.trim().split('-')[0].toLowerCase(), q: q ? parseFloat(q) : 1 };
    })
    .sort((a, b) => b.q - a.q)
    .map((item) => item.lang);
}

export const onRequest = defineMiddleware(async ({ request, url, redirect, cookies }, next) => {
  const { pathname } = url;

  // Skip: already on /en, static assets, API routes
  if (pathname.startsWith('/en') || SKIP_PREFIXES.some((p) => pathname.startsWith(p))) {
    return next();
  }

  // Skip: user has manually chosen a locale
  if (cookies.has('locale_preference')) {
    return next();
  }

  // Check Accept-Language header
  const acceptLang = request.headers.get('accept-language');
  if (!acceptLang) {
    return next();
  }

  const languages = parseAcceptLanguage(acceptLang);
  const prefersJapanese = languages.length === 0 || languages[0] === 'ja';

  if (prefersJapanese) {
    return next();
  }

  // Redirect non-Japanese users to /en version
  const enPath = `/en${pathname === '/' ? '' : pathname}`;
  return redirect(enPath, 302);
});