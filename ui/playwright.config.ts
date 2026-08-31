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
 * - WiFi troubleshooting
 * - Speed testing
 * - WebSocket connectivity
 *
 * Browsers: Chromium, Firefox, WebKit (Safari), Edge
 * Viewports: Desktop, Tablet, Mobile
 */
/**
 * The suite needs a running seed daemon, not a bare dev server, so there is no
 * sensible default to fall back to. Failing here with the command to run beats
 * pointing every spec at a port nothing is listening on and reporting a wall
 * of "element(s) not found".
 */
function requireBaseURL(): string {
  const fromEnv = process.env.E2E_BASE_URL;
  if (fromEnv) {
    return fromEnv;
  }
  throw new Error(
    'E2E_BASE_URL is not set. Run the suite through ./scripts/run-e2e.sh, which ' +
      'builds seed, starts it on a free port and exports E2E_BASE_URL:\n\n' +
      '  ./scripts/run-e2e.sh --project=chromium\n\n' +
      'To use a daemon you already have running, set it yourself:\n\n' +
      '  E2E_BASE_URL=https://127.0.0.1:8443 npx playwright test\n',
  );
}

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
  workers: process.env.CI ? 4 : 1,
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
    baseURL: requireBaseURL(),
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
  // No webServer. There used to be one starting `npm run dev` and waiting on
  // http://localhost:5173 — a URL nothing ever listened on, since vite.config
  // pins the dev server to port 3000. It could only ever time out after two
  // minutes, and it would not have worked at the right port either: the dev
  // server proxies nothing to the backend, while global-setup calls
  // /api/v1/setup/status against a real seed daemon.
  //
  // scripts/run-e2e.sh is the entry point. It builds the frontend and backend,
  // starts seed on a free port, waits for /__version, and exports
  // E2E_BASE_URL — which is why CI never hit any of this.
});
