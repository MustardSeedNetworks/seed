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

// Note: NOT tagged @smoke — assertions currently fragile (heading text drift,
// invalid-creds error path inconsistent). Tag once stabilised in follow-up.
test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any stored tokens
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();
  });

  test('should display login form when not authenticated', async ({ page }) => {
    await page.goto('/');

    // Check for login form elements
    await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
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

    // Should show error message
    await expect(page.getByText(/invalid|incorrect|failed/i)).toBeVisible({
      timeout: 5000,
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

    // Should redirect to dashboard
    await expect(page.getByRole('heading', { name: /link|dashboard/i })).toBeVisible({
      timeout: 10000,
    });
  });
});
