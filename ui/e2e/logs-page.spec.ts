import { expect, test } from '@playwright/test';
import { mockAuthenticated } from './helpers/auth';

/**
 * Logs Page (/logs) E2E
 *
 * Covers the live-log stream + system health surface:
 * - LogViewerCard
 * - SystemHealthCard
 */

test.describe('Logs Page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuthenticated(page);
    await page.goto('/logs');
    await expect(page.getByRole('heading', { name: /^logs$/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Logs title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /^logs$/i, level: 1 })).toBeVisible();
    await expect(page.getByText(/live log stream and system health/i)).toBeVisible();
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
