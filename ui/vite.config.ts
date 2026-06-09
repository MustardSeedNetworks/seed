/**
 * Vite Build Configuration
 *
 * Purpose: Configures the Vite development server and build process for the The Seed web frontend.
 * Handles bundling, module resolution, and development server settings.
 *
 * Configuration:
 * - React plugin: Enables JSX/TSX transformation and fast refresh during development
 * - Path alias: @ resolves to src/ directory for cleaner imports
 * - Dev server: Runs on port 3000 with HMR support
 * - Build output: Compiled directly to ../internal/api/ui for Go embed
 * - Embedding: Compiled frontend is embedded in Go binary via //go:embed directive
 *
 * Build Process:
 * 1. TypeScript compilation and bundling
 * 2. CSS processing and minification
 * 3. Asset optimization and tree-shaking
 * 4. Source map generation for production debugging
 * 5. Output to ../internal/api/ui for Go embedding (single source of truth)
 *
 * Usage:
 * ```bash
 * npm run dev     # Start dev server on port 3000
 * npm run build   # Build for production
 * npm run preview # Preview production build locally
 * ```
 *
 * Dependencies: vite, @vitejs/plugin-react
 * See: internal/api/embed_ui.go for how the build is embedded in the Go binary
 */

import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig, loadEnv, type PluginOption } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const analyze = env.ANALYZE === 'true';

  return {
    plugins: [
      react(),
      // Bundle treemap when ANALYZE=true (`npm run build:analyze`). Parity with
      // niac/stem; output lands in the gitignored ui/dist/ so it never ships.
      analyze &&
        (visualizer({
          open: true,
          filename: 'dist/bundle-stats.html',
          gzipSize: true,
        }) as PluginOption),
    ],
    resolve: {
      alias: [
        { find: /^@\//, replacement: fileURLToPath(new URL('./src/', import.meta.url)) },
        {
          find: /^@locales\//,
          replacement: fileURLToPath(new URL('../internal/i18n/locales/', import.meta.url)),
        },
      ],
      // Force a single copy of these so duplicate transitive versions don't
      // bloat the bundle or break React's single-instance invariants.
      dedupe: [
        'react',
        'react-dom',
        'react-router-dom',
        'lucide-react',
        'i18next',
        'react-i18next',
      ],
    },
    server: {
      port: 3000,
    },
    build: {
      // Output directly into the Go embed directory — no copying or syncing.
      // Canonical path shared with niac and stem: internal/api/ui/.
      // emptyOutDir intentionally omitted: outDir is outside Vite's project
      // root, so Vite defaults to false and preserves the tracked .gitkeep
      // placeholder (CLAUDE.md mandate).
      outDir: '../internal/api/ui',
      sourcemap: true,
      // Modern browser target — matches niac/stem. ES2022 covers all evergreen
      // browsers from 2023+; we don't support IE/legacy Safari.
      target: 'es2022',
      // CSS code splitting: allow per-route CSS bundles for better caching.
      cssCodeSplit: true,
      // Module preload polyfill: not needed for evergreen browsers (ES2022 target).
      modulePreload: { polyfill: false },
      // Real budget, not a cover-up. The shell stays the largest chunk; route
      // pages already lazy-load. Tighten toward niac's 350 as the shell shrinks.
      chunkSizeWarningLimit: 500,
      // Never inline assets as data: URLs (Vite default is 4096 bytes). Required
      // because @fontsource-variable ships small metric-override shim fonts that
      // would otherwise be inlined and violate the production `font-src 'self'`
      // CSP. With this set to 0, every asset bundles as a file under /assets/,
      // served from same-origin and properly HTTP-cacheable.
      assetsInlineLimit: 0,
      rollupOptions: {
        output: {
          // Split stable third-party deps into long-lived vendor chunks so an
          // app-code change doesn't bust their browser cache. jszip is left out
          // deliberately — it's export-only and should stay a lazy async chunk.
          manualChunks: (id: string) => {
            if (
              id.includes('/node_modules/react/') ||
              id.includes('/node_modules/react-dom/') ||
              id.includes('/node_modules/react-router-dom/') ||
              id.includes('/node_modules/scheduler/')
            )
              return 'vendor-react';
            if (id.includes('/node_modules/@tanstack/react-query/')) return 'vendor-query';
            if (
              id.includes('/node_modules/i18next/') ||
              id.includes('/node_modules/react-i18next/') ||
              id.includes('/node_modules/i18next-browser-languagedetector/')
            )
              return 'vendor-i18n';
            if (id.includes('/node_modules/zustand/') || id.includes('/node_modules/immer/'))
              return 'vendor-state';
            if (
              id.includes('/node_modules/lucide-react/') ||
              id.includes('/node_modules/tailwind-merge/')
            )
              return 'vendor-ui';
            return undefined;
          },
        },
      },
    },
  };
});
