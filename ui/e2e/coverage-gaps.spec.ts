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
    // profile-manage-open testid is on ProfileSelector.tsx (the
    // "Manage Profiles" link inside the profile dropdown).
    // profile-modal-close is on ProfileManagement.tsx.
    // Previously matched /manage profiles/i and /close/i — both
    // translated under es ("Administrar perfiles", "Cerrar").
    await page.getByLabel(/select profile/i).click();
    await page.getByTestId('profile-manage-open').click();

    // ProfileManagement.tsx already gives the H2 id="profile-modal-title"
    // (aria-labelledby target on the dialog); using it avoids i18n drift.
    await expect(page.locator('#profile-modal-title')).toBeVisible();

    await page.getByTestId('profile-modal-close').click();
  });

  test('opens log viewer modal', async ({ page }) => {
    // Card.tsx generates id="card-title-<slug>"; logs-card-maximize is
    // on LogViewerCard.tsx. Previously matched /full screen/i which
    // would miss under es ("Pantalla completa").
    const logsCardTitle = page.locator('#card-title-system-logs');
    await expect(logsCardTitle).toBeVisible();

    const logsCard = logsCardTitle.locator('..').first();
    await logsCard.getByTestId('logs-card-maximize').click();
    await expect(page.getByText(/system logs/i)).toBeVisible();
  });

  test('opens discovery modal', async ({ page }) => {
    // discovery-card-maximize is on NetworkDiscoveryCard.tsx (the
    // expand-to-modal button). Previously /open full screen view/i.
    await page.getByTestId('discovery-card-maximize').click();
    await expect(page.getByText(/network discovery/i)).toBeVisible();
  });
});
