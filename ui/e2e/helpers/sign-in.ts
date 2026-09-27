import type { BrowserContext } from '@playwright/test';

import { TEST_CREDENTIALS } from './auth.ts';

/**
 * Sign a fresh seed daemon in and persist the session as a Playwright
 * storageState. Two callers share it so they cannot drift apart:
 * e2e/global-setup.ts (the suite) and e2e/phone-width-sign-in.ts (the
 * fleet's 390px gate, which drives its own Playwright and runs this file
 * under plain `node` — hence the explicit `.ts` import above).
 *
 * **Cookie scoping (#1165):** the auth cookies are issued with
 * Secure=true + SameSite=Strict + HttpOnly=true (see
 * internal/auth/cookie.go). Earlier versions ran setup + login through
 * `request.newContext()` (Node-side HTTP client) and then opened a separate
 * `chromium` context to plant a localStorage flag — the persisted
 * storageState mixed cookies from two different contexts and the test
 * workers ended up authenticated only some of the time. Worse, the second
 * `context.storageState({ path })` call **overwrote** the first save with
 * whatever the SPA had after its mount-time /api/v1/status probe, which
 * sometimes already cleared the cookies because the status check raced the
 * cookie load.
 *
 * So everything happens in the one browser context the caller passes:
 *   1. complete first-run setup (via the context's own request client, so
 *      its cookies land in the same jar) if needed
 *   2. POST /api/v1/auth/login the same way
 *   3. navigate to "/" so the SPA mounts and the useAuth status check
 *      sees the cookies
 *   4. plant the seed.authenticated localStorage flag the SPA expects
 *   5. persist storageState ONCE at the end
 */
export async function signInAndPersist(context: BrowserContext, outPath: string): Promise<void> {
  await ensureSetupCompleted(context);

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
      `sign-in: /api/v1/auth/login returned ${loginResponse.status()}: ${body.slice(0, 200)}`,
    );
  }

  // If this ever needs an init script, register it with
  // `context.addInitScript()` BEFORE this line — never with
  // `page.addInitScript()` afterwards. On WebKit with @playwright/test 1.62.1,
  // an init script added to a page created by `context.newPage()` silently
  // never runs: no error, no warning, the callback simply does not execute
  // (seed#2286). The fixture `page` in a spec is not affected, which is why
  // every `addInitScript` in e2e/ is on a fixture page and this one is not
  // there at all.
  const page = await context.newPage();
  try {
    await page.goto('/');
    // The SPA should treat us as logged in. If /api/v1/status returns 401
    // here, the cookies aren't being sent and persisting state would
    // silently produce a broken setup file.
    const statusResponse = await page.request.get('/api/v1/status');
    if (!statusResponse.ok()) {
      throw new Error(
        `sign-in: post-login /api/v1/status returned ${statusResponse.status()} ` +
          `— cookies not being sent. See seed#1165 for the diagnosis history.`,
      );
    }

    // Some legacy code paths in the SPA check this flag to short-circuit
    // the login-form render before the async auth probe resolves.
    await page.evaluate(() => {
      window.localStorage.setItem('seed.authenticated', 'true');
    });
  } finally {
    await page.close();
  }

  await context.storageState({ path: outPath });
}

/**
 * On a fresh binary the admin password is unset and /api/v1/setup/status
 * reports needsSetup=true with a one-time setupToken; POST the token plus
 * the suite's well-known password to /api/v1/setup/complete. Idempotent.
 */
async function ensureSetupCompleted(context: BrowserContext): Promise<void> {
  const api = context.request;
  const statusResponse = await api.get('/api/v1/setup/status');
  if (!statusResponse.ok()) {
    const body = await statusResponse.text();
    throw new Error(
      `sign-in: /api/v1/setup/status returned ${statusResponse.status()}: ${body.slice(0, 200)}`,
    );
  }
  const status = (await statusResponse.json()) as {
    needsSetup: boolean;
    setupToken?: string;
  };
  if (!status.needsSetup) {
    return;
  }
  if (!status.setupToken) {
    throw new Error('sign-in: setup status reports needsSetup=true but no setupToken returned');
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
      `sign-in: /api/v1/setup/complete returned ${completeResponse.status()}: ${body.slice(0, 200)}`,
    );
  }
}
