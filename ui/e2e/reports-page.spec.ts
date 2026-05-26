import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Reports Page (/reports) E2E
 *
 * Covers the harvest module's reporting surface:
 * - SLADashboardCard renders
 */

test.describe('Reports Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/reports');
    await expect(page.getByRole('heading', { name: /^reports$/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Reports title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /^reports$/i, level: 1 })).toBeVisible();
    await expect(
      page.getByText(/aggregated sla dashboard|compliance|historical reporting/i),
    ).toBeVisible();
  });

  test('should land on the /reports route', async ({ page }) => {
    await expect(page).toHaveURL(/\/reports$/);
  });

  test('should render the SLA Dashboard card', async ({ page }) => {
    await expect(page.locator('text=/sla|dashboard|service.*level|target/i').first()).toBeVisible({
      timeout: 5000,
    });
  });
});
