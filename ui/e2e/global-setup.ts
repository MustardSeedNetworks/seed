import { mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { type BrowserContext, chromium, type FullConfig } from '@playwright/test';

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
 *
 * **Cookie scoping (#1165):** the auth cookies are issued with
 * Secure=true + SameSite=Strict + HttpOnly=true (see
 * internal/auth/cookie.go). Earlier versions of this script ran setup
 * + login through `request.newContext()` (Node-side HTTP client) and
 * then opened a separate `chromium` context to plant a localStorage
 * flag — the persisted storageState mixed cookies from two different
 * contexts and the test workers ended up authenticated only some of
 * the time. Worse, the second `context.storageState({ path })` call
 * **overwrote** the first save with whatever the SPA had after its
 * mount-time /api/v1/status probe, which sometimes already cleared
 * the cookies because the status check raced the cookie load.
 *
 * The current shape does everything in a single chromium context:
 *   1. open browser, create context
 *   2. complete first-run setup wizard (via apiContext bound to the
 *      same context's cookie jar) if needed
 *   3. POST /api/v1/auth/login via the same apiContext; cookies land
 *      in the browser's jar with the right Secure/SameSite scope
 *   4. navigate to "/" so the SPA mounts and the useAuth status check
 *      sees the cookies (this also seats any additional cookies the
 *      SPA might set, e.g. CSRF)
 *   5. plant the seed.authenticated localStorage flag the SPA expects
 *   6. persist storageState ONCE at the end
 *
 * The persisted file is identical to what a real browser would have
 * after a manual login, eliminating cross-context cookie scoping
 * bugs that previously caused ~85 of 113 E2E tests to fail on main.
 */
async function globalSetup(config: FullConfig): Promise<void> {
  const [project] = config.projects;
  if (project === undefined) {
    throw new Error('global-setup: no Playwright project configured');
  }
  const baseURL = project.use.baseURL ?? process.env.E2E_BASE_URL ?? 'http://localhost:5173';
  const outPath = resolve(__dirname, '..', AUTH_STORAGE_STATE);

  await mkdir(dirname(outPath), { recursive: true });

  const browser = await chromium.launch();
  try {
    const context = await browser.newContext({
      baseURL,
      ignoreHTTPSErrors: true,
    });
    try {
      await ensureSetupCompleted(context);
      await loginAndPersist(context, baseURL, outPath);
    } finally {
      await context.close();
    }
  } finally {
    await browser.close();
  }
}

/**
 * loginAndPersist runs the login flow through the same browser
 * context that will later be used to navigate the SPA, then writes
 * the resulting storageState. Doing everything in one context is the
 * difference between "cookies present but rejected by workers" and
 * "test workers authenticate cleanly" — see file-level comment.
 */
async function loginAndPersist(
  context: BrowserContext,
  baseURL: string,
  outPath: string,
): Promise<void> {
  const loginResponse = await context.request.post('/api/v1/auth/login', {
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

  // Navigate to "/" so the SPA mounts. This also lets the useAuth
  // hook's mount-time /api/v1/status probe land while the cookies are
  // valid — if we skipped this step a worker that opens the page
  // before any other request might race the cookie load.
  const page = await context.newPage();
  try {
    await page.goto('/');
    // Sanity check: the SPA should treat us as logged in. If
    // /api/v1/status returns 401 here, the cookies aren't being sent
    // and persisting state would silently produce a broken setup
    // file. Loud failure beats quietly-broken workers.
    const statusResponse = await page.request.get('/api/v1/status');
    if (!statusResponse.ok()) {
      throw new Error(
        `global-setup: post-login /api/v1/status returned ${statusResponse.status()} ` +
          `— cookies not being sent. baseURL=${baseURL}. ` +
          `See seed#1165 for the diagnosis history.`,
      );
    }

    // Some legacy code paths in the SPA check this flag to short-
    // circuit the login-form render before the async auth probe
    // resolves. Harmless on modern paths.
    await page.evaluate(() => {
      window.localStorage.setItem('seed.authenticated', 'true');
    });
  } finally {
    await page.close();
  }

  // Single persist at the end — no overwrite hazard.
  await context.storageState({ path: outPath });
}

/**
 * Bring the seed backend into a logged-in-able state. On a fresh
 * binary the admin password is unset and /api/v1/setup/status reports
 * needsSetup=true with a one-time setupToken; POST the token plus the
 * suite's well-known password to /api/v1/setup/complete. Idempotent
 * — if setup is already complete the function returns without
 * touching anything.
 *
 * Accepts a BrowserContext (rather than the bare APIRequestContext
 * the earlier shape used) so the setup-complete cookies land in the
 * same jar that loginAndPersist + the test workers will share.
 */
async function ensureSetupCompleted(context: BrowserContext): Promise<void> {
  const api = context.request;
  const statusResponse = await api.get('/api/v1/setup/status');
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

  const completeResponse = await api.post('/api/v1/setup/complete', {
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
