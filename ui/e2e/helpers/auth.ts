import { expect, type Locator, type Page } from '@playwright/test';

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
 * skipSetupWizard() still exists for specs that need to short-
 * circuit the first-run setup wizard without driving the form — the
 * storageState already covers auth, so this helper now only mocks
 * /api/setup/status. Kept as a single call site so future setup-
 * surface changes only require one edit.
 */

/**
 * Credentials the suite provisions in global-setup.ts.
 *
 * Wave 2 (#1047) enforces Argon2id + zxcvbn + HIBP on every password,
 * so the legacy "seed" literal would be rejected by the setup wizard
 * and login would never succeed. This value is long enough to clear
 * zxcvbn's score-3 threshold and synthetic enough to stay out of the
 * HIBP breach corpus; auth.spec.ts and auth-complete.spec.ts import
 * this constant directly so the password is in one place.
 */
export const TEST_CREDENTIALS = {
  username: 'admin',
  password: 'Seed-E2E-Strong-Passphrase-2026!', // gitleaks:allow — test fixture, provisioned in global-setup.ts
} as const;

/** Path (relative to ui/) where global-setup persists the storage state. */
export const AUTH_STORAGE_STATE = 'playwright/.auth/user.json';

/**
 * Skip the first-run setup wizard. Authentication is already handled
 * by the shared storageState wired up in playwright.config.ts.
 *
 * Must be called before page.goto so the route handler is registered.
 */
export async function skipSetupWizard(page: Page): Promise<void> {
  // Match both legacy /api/setup/status and v1-prefixed /api/v1/setup/status.
  // UI calls the v1 form; the legacy form is kept for resilience until any
  // remaining legacy callers are excised.
  await page.route(/\/api(\/v1)?\/setup\/status$/, (route) => {
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
  // Match both legacy /api/setup/status and v1-prefixed /api/v1/setup/status.
  // UI calls the v1 form; the legacy form is kept for resilience until any
  // remaining legacy callers are excised.
  await page.route(/\/api(\/v1)?\/setup\/status$/, (route) => {
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
  await page.getByTestId('login-submit').click();
}

/**
 * Fill the login form, submit, and wait for the authenticated
 * dashboard to mount (page-header-title). The caller is responsible
 * for navigating to the login surface (`await page.goto('/')`) and any
 * pre-login setup (viewport, request listeners, route mocks) first —
 * this helper owns only the credential entry, submit, and the
 * post-login settle.
 *
 * The 20s settle (vs. the per-test default 10s) is deliberate: the
 * post-login chain — login POST → SPA route swap → dashboard's first
 * data fetch → header paint — occasionally exceeds 10s on the heaviest
 * shard (auth-complete.spec.ts lands on shard-1) when the single CI
 * backend is serving four Playwright workers at once. That contention
 * produced a rotating one-test-per-run flake on main (#1170): whichever
 * login-then-dashboard test happened to land the slowest mount failed
 * the 10s assertion, then passed on retry. 20s sits comfortably inside
 * the 30s per-test budget (45s for the multi-login describes) and
 * removes the race without weakening the assertion — the login still
 * has to succeed and the real dashboard still has to render.
 */
export async function loginAndAwaitDashboard(
  page: Page,
  creds: { username: string; password: string } = TEST_CREDENTIALS,
): Promise<void> {
  await page.getByLabel(/username/i).fill(creds.username);
  await page.getByLabel(/password/i).fill(creds.password);
  await page.getByTestId('login-submit').click();
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
}

/**
 * Settings / Help live in the sidebar footer, not the header (Phase 2 —
 * see components/app/HeaderBar.tsx and the sidebar's FooterIconButton).
 * The buttons carry the hardcoded English aria-labels "Open settings" /
 * "Open help" (not i18n keys), so getByRole on the accessible name is
 * stable across locales.
 *
 * The shell renders the sidebar twice — a mobile drawer (`lg:hidden`)
 * and a desktop rail (`hidden lg:flex`). Only one is in the a11y tree at
 * a time (the other is display:none), but `.filter({ visible: true })`
 * is kept as a guard. Below the lg breakpoint (1024px) the sidebar is a
 * drawer behind the hamburger — call revealSidebar() first there.
 */
export function sidebarSettingsButton(page: Page): Locator {
  return page.getByRole('button', { name: 'Open settings' }).filter({ visible: true });
}

export function sidebarHelpButton(page: Page): Locator {
  return page.getByRole('button', { name: 'Open help' }).filter({ visible: true });
}

/**
 * Disable CSS transitions/animations for the page. The settings-drawer
 * open + accordion-expand transitions otherwise race Playwright's
 * scroll-into-view under parallel CI load, hanging clicks on deep
 * elements (e.g. the theme-toggle) until the 30s test timeout. Must be
 * called before page.goto so the style is installed on every navigation.
 */
export async function disableAnimations(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const style = document.createElement('style');
    style.textContent =
      '*,*::before,*::after{transition:none!important;animation:none!important;scroll-behavior:auto!important}';
    document.documentElement.appendChild(style);
  });
}

/**
 * Open the mobile sidebar drawer when below the lg breakpoint. The
 * hamburger ("Open menu") only exists under lg, so this is a no-op on
 * desktop viewports where the sidebar rail is always present.
 */
export async function revealSidebar(page: Page): Promise<void> {
  const menuButton = page.getByRole('button', { name: 'Open menu' });
  if (await menuButton.isVisible().catch(() => false)) {
    await menuButton.click();
  }
}
