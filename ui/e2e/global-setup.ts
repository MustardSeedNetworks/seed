import { mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, type FullConfig, request } from '@playwright/test';

import { AUTH_STORAGE_STATE, TEST_CREDENTIALS } from './helpers/auth';

// ESM equivalent of __dirname; Playwright executes this file as ESM so
// the CommonJS globals are not available.
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 * One real login at suite start; every spec shares the resulting
 * storageState (cookies + localStorage) via use.storageState in
 * playwright.config.ts. This collapses ~436 login attempts down to
 * exactly 1 — the per-IP login limiter
 * (internal/api/ratelimit.go: defaultMaxAttempts = 5 / 15 min) is no
 * longer a suite-wide cliff. auth.spec.ts and auth-complete.spec.ts
 * opt back into a clean unauthenticated context with
 * `test.use({ storageState: { cookies: [], origins: [] } })` so they
 * still exercise the real form.
 */
async function globalSetup(config: FullConfig): Promise<void> {
  const [project] = config.projects;
  if (project === undefined) {
    throw new Error('global-setup: no Playwright project configured');
  }
  const baseURL = project.use.baseURL ?? process.env.E2E_BASE_URL ?? 'http://localhost:5173';
  const outPath = resolve(__dirname, '..', AUTH_STORAGE_STATE);

  await mkdir(dirname(outPath), { recursive: true });

  // POST directly to /api/auth/login instead of driving the form so
  // global-setup does not need a full browser context just to capture
  // a single cookie. The server sets the session cookie on the
  // response; we hand the resulting cookies + an `authenticated` flag
  // in localStorage to Playwright as the persisted storage state.
  const apiContext = await request.newContext({
    baseURL,
    ignoreHTTPSErrors: true,
  });
  const response = await apiContext.post('/api/auth/login', {
    headers: { 'Content-Type': 'application/json' },
    data: {
      username: TEST_CREDENTIALS.username,
      password: TEST_CREDENTIALS.password,
    },
  });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(
      `global-setup: /api/auth/login returned ${response.status()}: ${body.slice(0, 200)}`,
    );
  }
  const state = await apiContext.storageState({ path: outPath });
  await apiContext.dispose();

  // Some specs use a browser-only side check (no inflight fetch) so we
  // also stash a "logged-in" flag in localStorage for the SPA origin.
  // We get an origins entry by quickly opening the baseURL in a
  // browser, writing the flag, and re-saving.
  const browser = await chromium.launch();
  const context = await browser.newContext({
    baseURL,
    ignoreHTTPSErrors: true,
    storageState: state,
  });
  const page = await context.newPage();
  await page.goto('/');
  await page.evaluate(() => {
    window.localStorage.setItem('seed.authenticated', 'true');
  });
  await context.storageState({ path: outPath });
  await browser.close();
}

export default globalSetup;
