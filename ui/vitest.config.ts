/**
 * Vitest Configuration
 *
 * Purpose: Configures the Vitest test framework and test environment for The Seed frontend.
 * Handles test discovery, environment setup, and coverage reporting.
 *
 * Configuration:
 * - Globals: Enable global test functions (describe, it, expect) without imports
 * - Environment: jsdom - Simulates browser DOM for React component testing
 * - Setup files: Loads test/setup.ts for global mocks and utilities
 * - File discovery: Matches *.test.ts and *.spec.tsx patterns (recursive)
 * - Coverage: V8 provider with multiple report formats (text, json, html, lcov)
 *
 * Test Execution:
 * 1. Setup file loads global mocks (localStorage, fetch, WebSocket)
 * 2. Test files are discovered and executed in jsdom environment
 * 3. Coverage data is collected and reported in multiple formats
 * 4. HTML reports generated to coverage/ directory
 *
 * Usage:
 * ```bash
 * npm test              # Run all tests
 * npm test -- --watch  # Run with file watching
 * npm test -- --coverage  # Generate coverage reports
 * npm test -- src/App.test.tsx  # Run specific test file
 * ```
 *
 * Coverage Goals:
 * - Exclude: test files, type definitions, config files, dist/
 * - Target: 80%+ line coverage on production code
 * - Reports: HTML at coverage/index.html, LCOV for CI/CD integration
 *
 * Dependencies: vitest, @vitejs/plugin-react, @vitest/ui (optional)
 */

import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@locales': fileURLToPath(new URL('../internal/i18n/locales', import.meta.url)),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      exclude: ['node_modules/', 'src/test/', '**/*.d.ts', '**/*.config.*', 'dist/'],
    },
  },
});
