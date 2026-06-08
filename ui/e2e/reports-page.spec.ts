import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Reports Page (/reports) E2E
 *
 * Covers the harvest module's reporting surface. Reports are gated behind the
 * `export_csv_json` feature (Starter tier or higher) via RequireFeature; the
 * E2E suite runs unlicensed (Free), so the page renders the license-gate
 * fallback, not the SLADashboardCard. The card-rendering path needs a licensed
 * fixture and is out of scope here.
 */

test.describe('Reports Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/reports');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Reports title', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await expect(page.getByTestId('page-header-description')).toBeVisible();
  });

  test('should land on the /reports route', async ({ page }) => {
    await expect(page).toHaveURL(/\/reports$/);
  });

  test('should gate reports behind a Starter+ license on the free tier', async ({ page }) => {
    // Unlicensed (Free) suite: RequireFeature renders the upsell fallback rather
    // than the SLADashboardCard. Scope to <main> so the assertion can never be
    // hijacked by sidebar nav labels (e.g. "Polling Targets" matches a loose
    // /target/i regex) — the prior text-regex `.first()` was brittle this way.
    await expect(page.getByRole('main').getByText(/reports require the starter tier/i)).toBeVisible(
      { timeout: 5000 },
    );
  });
});
