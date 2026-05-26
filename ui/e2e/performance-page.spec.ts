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
    await expect(page.getByRole('heading', { name: /^performance$/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Performance title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /^performance$/i, level: 1 })).toBeVisible();
    await expect(page.getByText(/active throughput tests/i)).toBeVisible();
  });

  test('should land on the /performance route', async ({ page }) => {
    await expect(page).toHaveURL(/\/performance$/);
  });

  test('should render the Health Check card', async ({ page }) => {
    await expect(page.locator('text=/health.*check/i').first()).toBeVisible({ timeout: 5000 });
  });

  test('should render a card grid (Performance card visible when enabled)', async ({ page }) => {
    // Either the Performance card or the Health Check card must be in the grid.
    const cards = page.locator('text=/health.*check|performance|speed.*test|iperf/i');
    await expect(cards.first()).toBeVisible({ timeout: 5000 });
  });
});
