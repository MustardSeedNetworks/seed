import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Licence feature gate (/path, /reports) E2E — seed#2688, seed#2669.
 *
 * Every licence-gated surface reads `features` off GET /api/v1/license
 * and nothing else. The endpoint never sent that field, so Pro and Trial
 * users were shown the Free upgrade gate on Path Analysis and Reports.
 *
 * The suite's daemon runs unlicensed and its activation state is the real
 * user config directory (Server.licenseDir has no CLI override), so a trial
 * cannot be started here without writing to the developer's own licence.
 * The paid tiers are driven by intercepting the endpoint with the exact
 * payload the Go handler now produces — internal/api's
 * TestLicenseStatusCarriesFeatures is what pins the server to that shape.
 *
 * Both gated routes are whole-page <GatedPreview> surfaces (#2669): on a tier
 * without the feature they render the pitch over a non-interactive sample, and
 * on a tier with it neither exists.
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
      await expect(main.getByTestId('gated-preview')).toHaveCount(0);
    });

    test(`renders Reports content on ${label}`, async ({ page }) => {
      await stubLicence(page, payload);
      await page.goto('/reports');
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

      const main = page.getByRole('main');
      await expect(main.getByTestId('card').first()).toBeVisible({ timeout: 10000 });
      await expect(main.getByTestId('gated-preview')).toHaveCount(0);
    });
  }

  // Every route the UI gates as a whole page, with what the sample must show.
  // A page that renders the pitch over an empty body is the defect #2669 fixed.
  const GATED_ROUTES = [
    { path: '/path', pitch: 'Pro feature: Path Analysis', sample: '203.0.113.24' },
    { path: '/reports', pitch: 'Starter feature: Reports', sample: 'Executive summary' },
  ] as const;

  for (const { path, pitch, sample } of GATED_ROUTES) {
    test(`previews the feature and pitches it on Free at ${path}`, async ({ page }) => {
      // No stub: the suite's own unlicensed daemon answers with `features: []`.
      await page.goto(path);
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

      const main = page.getByRole('main');
      await expect(main.getByTestId('gated-pitch')).toContainText(pitch, { timeout: 5000 });
      // The sample is the feature's own rendering, not an empty body.
      await expect(main.getByTestId('gated-preview')).toContainText(sample);
      await expect(main.getByText('seed license trial')).toBeVisible();
    });

    test(`leaves the sample unfocusable at ${path}`, async ({ page }) => {
      await page.goto(path);
      await expect(page.getByTestId('gated-preview')).toBeVisible({ timeout: 10000 });

      // tabIndex is a property `inert` does not change, so reading it proves
      // nothing: ask the engine instead — an inert control cannot take focus.
      const focusable = await page
        .getByTestId('gated-preview')
        .locator('[inert] button, [inert] a, [inert] input, [inert] select')
        .evaluateAll(
          (nodes) =>
            nodes.filter((node) => {
              (node as HTMLElement).focus();
              return document.activeElement === node;
            }).length,
        );
      expect(focusable).toBe(0);
    });
  }
});
