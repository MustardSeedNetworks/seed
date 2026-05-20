import type { Page } from '@playwright/test';

/**
 * Shared E2E auth helpers.
 *
 * Wave 2 (#1047) tightened the login limiter to defaultMaxAttempts = 5
 * per 15-minute window per IP (internal/api/ratelimit.go). With ~30
 * specs and ~436 tests each driving the real /api/auth/login in
 * beforeEach, the suite blew past the budget after a handful of tests
 * and subsequent specs landed on a lockout.
 *
 * The fix is a single real login in e2e/global-setup.ts, persisted to
 * AUTH_STORAGE_STATE and shared across every spec via Playwright's
 * use.storageState. This collapses ~436 login attempts down to 1.
 *
 * Specs that genuinely test the auth flow (auth.spec.ts,
 * auth-complete.spec.ts) opt out of the shared state with
 * `test.use({ storageState: { cookies: [], origins: [] } })` and use
 * loginViaUI() / their own flows.
 *
 * mockAuthenticated() still exists for specs that need to short-
 * circuit the first-run setup wizard without driving the form — the
 * storageState already covers auth, so this helper now only mocks
 * /api/setup/status. Kept as a single call site so future setup-
 * surface changes only require one edit.
 */

export const TEST_CREDENTIALS = {
  username: 'admin',
  password: 'seed',
} as const;

/** Path (relative to ui/) where global-setup persists the storage state. */
export const AUTH_STORAGE_STATE = 'playwright/.auth/user.json';

/**
 * Skip the first-run setup wizard. Authentication is already handled
 * by the shared storageState wired up in playwright.config.ts.
 *
 * Must be called before page.goto so the route handler is registered.
 */
export async function mockAuthenticated(page: Page): Promise<void> {
  await page.route('**/api/setup/status', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ needsSetup: false, username: 'admin' }),
    });
  });
}

/**
 * Drive the real login form. Reserve for specs that genuinely test
 * the auth flow end-to-end (auth.spec.ts, auth-complete.spec.ts).
 * Other specs inherit auth via the global-setup storageState.
 */
export async function loginViaUI(
  page: Page,
  creds: { username: string; password: string } = TEST_CREDENTIALS,
): Promise<void> {
  await page.route('**/api/setup/status', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ needsSetup: false, username: 'admin' }),
    });
  });
  await page.goto('/');
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await page.getByLabel(/username/i).fill(creds.username);
  await page.getByLabel(/password/i).fill(creds.password);
  await page.getByRole('button', { name: /sign in|login/i }).click();
}
