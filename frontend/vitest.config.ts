/// <reference types="vitest/globals" />
/// <reference types="node" />
import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
  plugins: [
    sveltekit({
      exclude: ['**/node_modules/**'],
    }),
  ],
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.ts'],
    exclude: ['src/routes/orchestration/webhooks/proxy-debug.test.ts'],
    setupFiles: ['src/vitest-setup.ts'],
  },
});
