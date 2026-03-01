import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import node from '@astrojs/node';

const { BACKEND_INTERNAL_URL = 'http://localhost:8080' } = process.env;

// https://astro.build/config
export default defineConfig({
  adapter: node({ mode: 'standalone' }),
  integrations: [react()],
  vite: {
    server: {
      host: true,
      watch: {
        usePolling: true,
        interval: 1000,
        ignored: ['**/node_modules/**', '**/dist/**', '**/.git/**'],
      },
      proxy: {
        '/api': {
          target: BACKEND_INTERNAL_URL,
          changeOrigin: true,
        },
        '/uploads': {
          target: BACKEND_INTERNAL_URL,
          changeOrigin: true,
        },
      },
    },
  },
  i18n: {
    locales: ['ja', 'en'],
    defaultLocale: 'ja',
    prefixDefaultLocale: false,
  },
});
