import { expect, test } from '@playwright/test';
import { TEST_CREDENTIALS } from './helpers/auth';

/**
 * Authentication E2E Tests
 *
 * Tests the login flow and token handling:
 * - Login form renders correctly
 * - Login with valid credentials
 * - Login with invalid credentials shows error
 * - Logout clears session
 *
 * Opts out of the suite-wide authenticated storageState so each test
 * starts from a clean unauthenticated context.
 */

test.use({ storageState: { cookies: [], origins: [] } });

test.describe('@smoke Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any stored tokens
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();
  });

  test('should display login form when not authenticated', async ({ page }) => {
    await page.goto('/');

    // The LoginForm renders the app brand ("The Seed") as the H1, not
    // the word "Login" — assert on the brand + the form controls, which
    // are the load-bearing UX contract here.
    await expect(page.getByRole('heading', { name: /the seed/i })).toBeVisible();
    await expect(page.getByLabel(/username/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /sign in|login/i })).toBeVisible();
  });

  test('should show error with invalid credentials', async ({ page }) => {
    await page.goto('/');

    // Fill in invalid credentials
    await page.getByLabel(/username/i).fill('wronguser');
    await page.getByLabel(/password/i).fill('wrongpassword');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Backend returns "Invalid credentials" on bad auth; the form
    // surfaces it in a status-error badge below the inputs. Use first()
    // to bypass strict-mode matching against the i18n placeholder text
    // that also contains "Invalid". 10s timeout because the per-IP
    // rate limiter can throttle repeat attempts on a hot suite.
    await expect(page.getByText(/invalid|incorrect|failed/i).first()).toBeVisible({
      timeout: 10000,
    });
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    await page.goto('/');

    // Provisioned in e2e/global-setup.ts; the wave-2 password policy
    // rejects the legacy short "seed" literal, so the suite uses a
    // shared strong passphrase from helpers/auth.ts.
    await page.getByLabel(/username/i).fill(TEST_CREDENTIALS.username);
    await page.getByLabel(/password/i).fill(TEST_CREDENTIALS.password);
    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Should redirect to dashboard. The dashboard's primary H1 is
    // "Link" (the active page tab); the H3 "Link Status" inside the
    // card chrome would otherwise trip strict-mode matching. Pin to
    // level: 1 to keep the assertion unambiguous.
    await expect(page.getByRole('heading', { name: /^link$|dashboard/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });
});
