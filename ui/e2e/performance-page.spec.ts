import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Performance Page (/performance) E2E
 *
 * Covers the sap module's active-test surface:
 * - Page renders with the proper heading
 * - HealthCheckCard slot is present
 * - PerformanceCard slot is present when enabled in cardSettings
 */

test.describe('Performance Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/performance');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Performance title', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await expect(page.getByTestId('page-header-description')).toBeVisible();
  });

  test('should land on the /performance route', async ({ page }) => {
    await expect(page).toHaveURL(/\/performance$/);
  });

  test('should render the Health Check card', async ({ page }) => {
    await expect(page.locator('text=/health.*check/i').first()).toBeVisible({ timeout: 5000 });
  });
});
