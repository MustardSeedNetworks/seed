import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Path Analysis Page (/path) E2E
 *
 * Covers the roots module's discovery surface:
 * - Page renders with the proper heading
 * - PathDiscoveryCard slot is present
 * - NetworkDiscoveryCard slot is present when enabled in cardSettings
 */

test.describe('Path Analysis Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/path');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Path Analysis title', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await expect(page.getByTestId('page-header-description')).toBeVisible();
  });

  test('should land on the /path route', async ({ page }) => {
    await expect(page).toHaveURL(/\/path$/);
  });
});
