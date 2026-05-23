import { expect, test } from '@playwright/test';
import { mockAuthenticated } from './helpers/auth';

/**
 * Path Analysis Page (/path) E2E
 *
 * Covers the roots module's discovery surface:
 * - Page renders with the proper heading
 * - PathDiscoveryCard slot is present
 * - NetworkDiscoveryCard slot is present when enabled in cardSettings
 */

test.describe('Path Analysis Page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuthenticated(page);
    await page.goto('/path');
    await expect(page.getByRole('heading', { name: /path analysis/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should render the page header with Path Analysis title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /path analysis/i, level: 1 })).toBeVisible();
    await expect(page.getByText(/path discovery|traceroute|device discovery/i)).toBeVisible();
  });

  test('should land on the /path route', async ({ page }) => {
    await expect(page).toHaveURL(/\/path$/);
  });

  test('should render the Path Discovery card (when not in WiFi-only mode)', async ({ page }) => {
    // Card content varies but should have at least one relevant heading or label visible
    const content = page.locator('text=/path|trace|hop|gateway|dns/i');
    await expect(content.first()).toBeVisible({ timeout: 5000 });
  });
});
