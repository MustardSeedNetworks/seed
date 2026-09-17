import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Licence feature gate (/path, /reports) E2E — seed#2688.
 *
 * Every <RequireFeature> surface reads `features` off GET /api/v1/license
 * and nothing else. The endpoint never sent that field, so Pro and Trial
 * users were shown the Free upgrade gate on Path Analysis and Reports.
 *
 * The suite's daemon runs unlicensed and its activation state is the real
 * user config directory (Server.licenseDir has no CLI override), so a trial
 * cannot be started here without writing to the developer's own licence.
 * The paid tiers are driven by intercepting the endpoint with the exact
 * payload the Go handler now produces — internal/api's
 * TestLicenseStatusCarriesFeatures is what pins the server to that shape.
 */

const PRO_FEATURES = [
  'export_csv_json',
  'dns_monitoring',
  'ssl_cert_monitoring',
  'path_analysis',
  'wifi_association_forensics',
  'wifi_management_capture',
  'rest_api',
];

async function stubLicence(page: Page, body: Record<string, unknown>): Promise<void> {
  await page.route('**/api/v1/license', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(body),
    });
  });
}

const trialPayload = {
  tier: 'Trial',
  tierValue: 2,
  isTrialMode: true,
  trialDaysLeft: 14,
  canMintTokens: true,
  activated: true,
  features: PRO_FEATURES,
};

const proPayload = {
  tier: 'Pro',
  tierValue: 2,
  isTrialMode: false,
  canMintTokens: true,
  activated: true,
  features: PRO_FEATURES,
};

test.describe('licence feature gate', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  for (const [label, payload] of [
    ['a trial', trialPayload],
    ['Pro', proPayload],
  ] as const) {
    test(`renders Path Analysis content on ${label}`, async ({ page }) => {
      await stubLicence(page, payload);
      await page.goto('/path');
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

      const main = page.getByRole('main');
      await expect(main.getByTestId('card').first()).toBeVisible({ timeout: 10000 });
      await expect(main.getByText(/Path Analysis is a Pro-tier feature/i)).toHaveCount(0);
    });

    test(`renders Reports content on ${label}`, async ({ page }) => {
      await stubLicence(page, payload);
      await page.goto('/reports');
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

      const main = page.getByRole('main');
      await expect(main.getByTestId('card').first()).toBeVisible({ timeout: 10000 });
      await expect(main.getByText(/Reports require the Starter tier/i)).toHaveCount(0);
    });
  }

  test('keeps the gate on Free', async ({ page }) => {
    // No stub: the suite's own unlicensed daemon answers, and the real
    // response now carries `features: []`.
    await page.goto('/path');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await expect(
      page.getByRole('main').getByText(/Path Analysis is a Pro-tier feature/i),
    ).toBeVisible({ timeout: 5000 });

    await page.goto('/reports');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('main').getByText(/Reports require the Starter tier/i)).toBeVisible(
      { timeout: 5000 },
    );
  });
});
