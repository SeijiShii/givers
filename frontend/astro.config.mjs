// @ts-check
import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import node from '@astrojs/node';

// https://astro.build/config
export default defineConfig({
  adapter: node({ mode: 'standalone' }),
  integrations: [react()],
  vite: {
    server: {
      host: true,
      watch: {
        usePolling: true,
      },
    },
  },
  i18n: {
    locales: ['ja', 'en'],
    defaultLocale: 'ja',
    prefixDefaultLocale: false,
  },
});
