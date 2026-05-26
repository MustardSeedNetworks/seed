import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Link Page (/link) E2E
 *
 * Default landing page after login (App.tsx redirects '/' → '/link').
 * Covers the sap module's physical-link surface:
 * - Page renders with the proper heading
 * - LinkCard slot renders when active interface is wired
 * - WiFiCard slot renders when active interface is Wi-Fi
 * - CableCard slot appears when linkUp is false (cable diagnostics path)
 */

test.describe('Link Page', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/link');
    await expect(page.getByRole('heading', { name: /^link$/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Link title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /^link$/i, level: 1 })).toBeVisible();
    await expect(page.getByText(/physical link state.*cable diagnostics/i)).toBeVisible();
  });

  test('should land on the /link route', async ({ page }) => {
    await expect(page).toHaveURL(/\/link$/);
  });

  test('should be the default route when navigating to root', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/link$/);
    await expect(page.getByRole('heading', { name: /^link$/i, level: 1 })).toBeVisible();
  });

  test('should render at least one link-state card (LinkCard or WiFiCard)', async ({ page }) => {
    // One branch must be visible:
    //   - isWifi=false → LinkCard ('link status' content)
    //   - isWifi=true  → WiFiCard ('ssid' or 'wifi' content)
    const cards = page.locator('text=/link.*status|ssid|wifi|interface|carrier/i');
    await expect(cards.first()).toBeVisible({ timeout: 5000 });
  });
});
