import { mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { type APIRequestContext, chromium, type FullConfig, request } from '@playwright/test';

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
 *
 * In CI the seed binary starts fresh each run, which means there is
 * no admin user — the initial-setup wizard must complete before
 * /api/v1/auth/login will accept any credentials. This script runs
 * the setup-then-login flow once and persists the resulting session
 * cookies as the suite's default storage state.
 */
async function globalSetup(config: FullConfig): Promise<void> {
  const [project] = config.projects;
  if (project === undefined) {
    throw new Error('global-setup: no Playwright project configured');
  }
  const baseURL = project.use.baseURL ?? process.env.E2E_BASE_URL ?? 'http://localhost:5173';
  const outPath = resolve(__dirname, '..', AUTH_STORAGE_STATE);

  await mkdir(dirname(outPath), { recursive: true });

  const apiContext = await request.newContext({
    baseURL,
    ignoreHTTPSErrors: true,
  });

  try {
    await ensureSetupCompleted(apiContext);

    const loginResponse = await apiContext.post('/api/v1/auth/login', {
      headers: { 'Content-Type': 'application/json' },
      data: {
        username: TEST_CREDENTIALS.username,
        password: TEST_CREDENTIALS.password,
      },
    });
    if (!loginResponse.ok()) {
      const body = await loginResponse.text();
      throw new Error(
        `global-setup: /api/v1/auth/login returned ${loginResponse.status()}: ${body.slice(0, 200)}`,
      );
    }

    await apiContext.storageState({ path: outPath });
  } finally {
    await apiContext.dispose();
  }

  // Attach a localStorage flag for the SPA origin so the in-app
  // useAuth() hook's mount-time check doesn't flip the UI back to the
  // login form before its /api/status probe lands.
  const browser = await chromium.launch();
  try {
    const context = await browser.newContext({
      baseURL,
      ignoreHTTPSErrors: true,
      storageState: outPath,
    });
    const page = await context.newPage();
    await page.goto('/');
    await page.evaluate(() => {
      window.localStorage.setItem('seed.authenticated', 'true');
    });
    await context.storageState({ path: outPath });
  } finally {
    await browser.close();
  }
}

/**
 * Bring the seed backend into a logged-in-able state. On a fresh
 * binary the admin password is unset and /api/v1/setup/status reports
 * needsSetup=true with a one-time setupToken; POST the token plus the
 * suite's well-known password to /api/v1/setup/complete. Idempotent
 * — if setup is already complete the function returns without
 * touching anything.
 */
async function ensureSetupCompleted(apiContext: APIRequestContext): Promise<void> {
  const statusResponse = await apiContext.get('/api/v1/setup/status');
  if (!statusResponse.ok()) {
    const body = await statusResponse.text();
    throw new Error(
      `global-setup: /api/v1/setup/status returned ${statusResponse.status()}: ${body.slice(0, 200)}`,
    );
  }
  const status = (await statusResponse.json()) as {
    needsSetup: boolean;
    setupToken?: string;
    username?: string;
  };
  if (!status.needsSetup) {
    return;
  }
  if (!status.setupToken) {
    throw new Error(
      'global-setup: setup status reports needsSetup=true but no setupToken returned',
    );
  }

  const completeResponse = await apiContext.post('/api/v1/setup/complete', {
    headers: { 'Content-Type': 'application/json' },
    data: {
      password: TEST_CREDENTIALS.password,
      setupToken: status.setupToken,
    },
  });
  if (!completeResponse.ok()) {
    const body = await completeResponse.text();
    throw new Error(
      `global-setup: /api/v1/setup/complete returned ${completeResponse.status()}: ${body.slice(0, 200)}`,
    );
  }
}

export default globalSetup;
