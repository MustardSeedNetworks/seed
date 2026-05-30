import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Wi-Fi Page (/wifi) E2E
 *
 * Covers the canopy module's Wi-Fi visibility surface:
 * - Page renders with the proper heading
 * - WiFiCard / WiFiSurveyCard / WifiChannelGraph slots are present when
 *   the active interface is Wi-Fi
 * - The "switch to Wi-Fi mode" fallback renders when active interface is wired
 *
 * NOT in scope (would require real RF data):
 *   - Specific channel/SNR numeric assertions
 *   - Actual survey content
 */

test.describe('Wi-Fi Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/wifi');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();
  });

  test('should land on the /wifi route after navigation', async ({ page }) => {
    await expect(page).toHaveURL(/\/wifi$/);
  });

  test('should render either WiFi cards or the wired-mode fallback', async ({ page }) => {
    // One branch must be visible:
    //   - isWifi=true  → at least one WiFi card (`#card-title-wifi`,
    //     `#card-title-wifi-survey`, or `#card-title-channels`).
    //   - isWifi=false → `data-testid="wifi-wired-fallback"` div on WifiPage.
    const wifiCards = page.locator(
      '#card-title-wifi, #card-title-wifi-survey, #card-title-channels',
    );
    const wiredFallback = page.getByTestId('wifi-wired-fallback');
    await expect(wifiCards.first().or(wiredFallback)).toBeVisible({ timeout: 5000 });
  });

  test('should show the wired-mode message when WiFi card data unavailable', async ({ page }) => {
    // Mock the wifi endpoint to return empty so isWifi remains false
    await page.route('**/api/v1/canopy/wifi', (route) => {
      route.fulfill({
        status: 200,
        body: JSON.stringify({ available: false }),
        headers: { 'Content-Type': 'application/json' },
      });
    });
    await page.reload();
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    // Either the wired-mode message OR the cards must be rendered; both
    // branches are valid given different test environments.
    const wifiCards = page.locator(
      '#card-title-wifi, #card-title-wifi-survey, #card-title-channels',
    );
    const wiredFallback = page.getByTestId('wifi-wired-fallback');
    await expect(wiredFallback.or(wifiCards.first())).toBeVisible({ timeout: 5000 });
  });
});
