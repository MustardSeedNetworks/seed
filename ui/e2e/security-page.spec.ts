import { expect, test } from '@playwright/test';
import { mockAuthenticated } from './helpers/auth';

/**
 * Security Page (/security) E2E
 *
 * Covers the shell module's security posture surface:
 * - Page renders with the proper heading
 * - MFA card and Guest Network Audit card slots are present
 */

test.describe('Security Page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuthenticated(page);
    await page.goto('/security');
    await expect(page.getByRole('heading', { name: /^security$/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Security title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /^security$/i, level: 1 })).toBeVisible();
    await expect(page.getByText(/guest network isolation audit/i)).toBeVisible();
  });

  test('should land on the /security route', async ({ page }) => {
    await expect(page).toHaveURL(/\/security$/);
  });

  test('should render the MFA card', async ({ page }) => {
    await expect(page.locator('text=/mfa|two.factor|2fa/i').first()).toBeVisible({
      timeout: 5000,
    });
  });

  test('should render the Guest Network Audit card', async ({ page }) => {
    await expect(page.locator('text=/guest.*network|guest.*audit/i').first()).toBeVisible({
      timeout: 5000,
    });
  });
});
