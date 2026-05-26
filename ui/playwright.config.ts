import { defineConfig, devices } from '@playwright/test';

import { AUTH_STORAGE_STATE } from './e2e/helpers/auth';

/**
 * Playwright E2E Test Configuration
 *
 * Comprehensive browser testing for critical user flows:
 * - Authentication (login/logout)
 * - Dashboard card rendering
 * - Settings save/load
 * - Network discovery
 * - WiFi survey
 * - Speed testing
 * - WebSocket connectivity
 *
 * Browsers: Chromium, Firefox, WebKit (Safari), Edge
 * Viewports: Desktop, Tablet, Mobile
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  // retries 1 (not 2) — one retry is enough to dodge transient flakes; the
  //   second retry was costing ~30s × N flaky tests with no incremental signal.
  // workers 2 in CI (was 1) — GH Actions runners are 4-vCPU; 1 worker wastes
  //   75% of the box and is the biggest single driver of the 30-min full-suite
  //   runtime we're trying to fix (issue #1072). Two workers run safely against
  //   our single-instance backend without state interference today; bump to 4
  //   once we've audited tests that mutate global server state.
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : undefined,
  timeout: 30000,
  expect: {
    timeout: 10000,
  },
  // Single real login at suite start; persisted to AUTH_STORAGE_STATE
  // and replayed into every test via use.storageState below. See
  // e2e/global-setup.ts and the comment in e2e/helpers/auth.ts.
  globalSetup: './e2e/global-setup.ts',
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
    ['json', { outputFile: 'playwright-report/results.json' }],
  ],
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
    ignoreHTTPSErrors: true,
    // Cookies + localStorage captured by global-setup. Specs that
    // need an unauthenticated context (auth.spec.ts,
    // auth-complete.spec.ts, setup-wizard.spec.ts) override with
    // test.use({ storageState: { cookies: [], origins: [] } }).
    storageState: AUTH_STORAGE_STATE,
  },
  projects: [
    // Per msn-docs-internal/05-Engineering/E2E_CONVENTIONS.md, only chromium
    // (covers Chrome and Edge — same engine) and webkit (covers Safari) are
    // supported. The previous firefox/edge/mobile-chrome/mobile-safari/tablet
    // entries were configured but never invoked in CI, lying about coverage.
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  // Run local dev server before tests if not in CI
  webServer: process.env.CI
    ? undefined
    : {
        command: 'npm run dev',
        url: 'http://localhost:5173',
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
      },
});
