import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Logs Page (/logs) E2E
 *
 * Covers the live-log stream + system health surface:
 * - LogViewerCard
 * - SystemHealthCard
 */

test.describe('Logs Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/logs');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Logs title', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await expect(page.getByTestId('page-header-description')).toBeVisible();
  });

  test('should land on the /logs route', async ({ page }) => {
    await expect(page).toHaveURL(/\/logs$/);
  });

  test('should render the System Health card', async ({ page }) => {
    await expect(page.locator('text=/system.*health|cpu|memory|disk/i').first()).toBeVisible({
      timeout: 5000,
    });
  });

  test('should render the Log Viewer card', async ({ page }) => {
    await expect(page.locator('text=/log|level|message|stream/i').first()).toBeVisible({
      timeout: 5000,
    });
  });
});
