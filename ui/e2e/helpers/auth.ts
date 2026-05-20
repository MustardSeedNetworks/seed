import type { Page } from '@playwright/test';

/**
 * Shared E2E auth helpers.
 *
 * Wave 2 (#1047) tightened the login limiter to defaultMaxAttempts = 5
 * per 15-minute window per IP (internal/api/ratelimit.go). With ~18
 * spec files and ~436 tests each driving the real /api/auth/login in
 * beforeEach, the suite blows through that budget after a handful of
 * tests and subsequent specs land on a lockout — the login form stays
 * mounted, shell clicks time out, and the whole E2E job runs out the
 * 30-minute timeout. Specs that aren't actually testing the auth flow
 * itself should call mockAuthenticated() instead of pounding the real
 * endpoint.
 */

export const TEST_CREDENTIALS = {
  username: 'admin',
  password: 'seed',
} as const;

/**
 * Skip the login form for tests whose subject is anything other than
 * the authentication flow itself.
 *
 * Mocks the setup-status probe so the first-run wizard doesn't fire,
 * mocks /api/status to return 200 so useAuth's mount-time session
 * check hydrates isAuthenticated=true without driving the real login
 * endpoint. Must be called before page.goto so the route handlers are
 * registered before any navigation.
 */
export async function mockAuthenticated(page: Page): Promise<void> {
  await page.route('**/api/setup/status', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ needsSetup: false, username: 'admin' }),
    });
  });
  await page.route('**/api/status', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ authenticated: true, username: 'admin' }),
    });
  });
}

/**
 * Drive the real login form. Reserve for specs that genuinely test
 * the auth flow end-to-end (auth.spec.ts, auth-complete.spec.ts).
 * Other specs should use mockAuthenticated() to avoid the rate-limit
 * cliff.
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
