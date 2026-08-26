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
import babel from '@rolldown/plugin-babel';
import react, { reactCompilerPreset } from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [
    react(),
    // The React Compiler, matching vite.config.ts. Without it the suite
    // exercises un-compiled components while the shipped bundle is compiled —
    // so a memo the compiler subsumes looks required here, and a compiler
    // regression could never fail a test.
    babel({ presets: [reactCompilerPreset()] }),
  ],
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
      // Anti-regression floor (set ~2pp below current measurement).
      // Ratchet up toward the CLAUDE.md mandatory minimum of 50% as
      // coverage improves. Current: lines 31, branches 17, functions
      // 22, stmts 31.
      // Ratchet only. Measured 2026-08-25 at statements 32.11 / branches 26.45
      // / functions 27.54 / lines 34.51, so each floor sits just under the real
      // number with a little margin for run-to-run drift.
      //
      // Branches had the most slack — a 14% floor against 26% actual meant
      // roughly half the branch coverage the suite already had could be
      // deleted without the gate noticing.
      //
      // Raise these when coverage rises; never lower them to make a change
      // pass.
      thresholds: {
        lines: 34,
        branches: 26,
        functions: 27,
        statements: 32,
      },
    },
  },
});
