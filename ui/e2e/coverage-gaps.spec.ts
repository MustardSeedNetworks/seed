import { expect, test } from '@playwright/test';
import { mockAuthenticated } from './helpers/auth';

test.describe('Coverage gaps', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuthenticated(page);
    await page.goto('/');
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('opens profile management modal', async ({ page }) => {
    await page.getByLabel(/select profile/i).click();
    await page.getByRole('button', { name: /manage profiles/i }).click();

    await expect(page.getByRole('heading', { name: /profile management/i })).toBeVisible();

    await page.getByRole('button', { name: /close/i }).click();
  });

  test('opens log viewer modal', async ({ page }) => {
    const logsCardTitle = page
      .locator('h3:has-text("System Logs"), h4:has-text("System Logs")')
      .first();
    await expect(logsCardTitle).toBeVisible();

    const logsCard = logsCardTitle.locator('..').first();
    await logsCard.getByRole('button', { name: /full screen/i }).click();
    await expect(page.getByText(/system logs/i)).toBeVisible();
  });

  test('opens discovery modal', async ({ page }) => {
    await page.getByRole('button', { name: /open full screen view/i }).click();
    await expect(page.getByText(/network discovery/i)).toBeVisible();
  });
});
