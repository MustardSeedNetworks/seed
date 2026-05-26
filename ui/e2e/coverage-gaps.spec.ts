import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

test.describe('Coverage gaps', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('opens profile management modal', async ({ page }) => {
    await page.getByLabel(/select profile/i).click();
    await page.getByRole('button', { name: /manage profiles/i }).click();

    // ProfileManagement.tsx already gives the H2 id="profile-modal-title"
    // (aria-labelledby target on the dialog); using it avoids i18n drift
    // on the /profile management/i regex.
    await expect(page.locator('#profile-modal-title')).toBeVisible();

    await page.getByRole('button', { name: /close/i }).click();
  });

  test('opens log viewer modal', async ({ page }) => {
    // Card.tsx generates id="card-title-<slug>" — see comment in
    // dashboard.spec.ts. "System Logs" → "system-logs".
    const logsCardTitle = page.locator('#card-title-system-logs');
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
