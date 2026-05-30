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
  // workers 4 in CI (bumped from 2 in PR-2.5) — GH Actions ubuntu-latest is
  //   4-vCPU; running 4 workers fills the box. The matrix in ci.yml further
  //   splits the suite across 4 shards per browser, so each runner sees only
  //   ~46 tests and the wall-clock per browser drops from ~17 min (workers=2,
  //   single shard) to ~5 min (workers=4, 4 shards) — and to ~2 min once the
  //   failing-test backlog from PRs 1–5 is cleared.
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 4 : undefined,
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
    // Gated to local dev only. CI is expected to provision a CA-trusted
    // cert (the seed binary's self-signed cert is fine for laptop work but
    // CI MUST enforce real TLS per E2E_CONVENTIONS). If a CI run needs the
    // self-signed fallback, set PLAYWRIGHT_IGNORE_HTTPS_ERRORS=true in the
    // workflow env — that override is honored below.
    ignoreHTTPSErrors: process.env.PLAYWRIGHT_IGNORE_HTTPS_ERRORS === 'true' || !process.env.CI,
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
