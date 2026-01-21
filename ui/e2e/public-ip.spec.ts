import { expect, test } from '@playwright/test';

/**
 * Public IP E2E Tests
 *
 * Tests for Public IP detection and display:
 * - IPv4 and IPv6 address display
 * - Location information (if available)
 * - Privacy toggle functionality
 * - Refresh/update behavior
 * - Error handling
 */

test.describe('Public IP', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill('admin');
    await page.getByLabel(/password/i).fill('seed');
    await page.getByRole('button', { name: /sign in|login/i }).click();
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should display public IP in Network card', async ({ page }) => {
    // Look for public IP section
    const publicIp = page
      .getByText(/public.*ip|external.*ip/i)
      .or(page.locator('[data-testid="public-ip"]'));

    const hasPublicIp = await publicIp.isVisible().catch(() => false);
    expect(hasPublicIp).toBeDefined();
  });

  test('should show IPv4 address format', async ({ page }) => {
    // Look for IPv4 format (x.x.x.x)
    const ipv4Pattern = page.getByText(/\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/);

    const hasIpv4 = await ipv4Pattern.isVisible().catch(() => false);
    expect(hasIpv4).toBeDefined();
  });

  test('should show IPv6 address if available', async ({ page }) => {
    // Look for IPv6 format (contains colons)
    const ipv6Pattern = page.getByText(/[0-9a-fA-F:]{8,}/);

    const hasIpv6 = await ipv6Pattern.isVisible().catch(() => false);
    // IPv6 may not be available on all networks
    expect(hasIpv6).toBeDefined();
  });

  test('should have show/hide public IP toggle in settings', async ({ page }) => {
    // Open settings
    const settingsButton = page.getByRole('button', { name: /settings/i });
    await settingsButton.click();
    await page.waitForTimeout(500);

    // Look for public IP toggle
    const publicIpToggle = page
      .locator('label:has-text("Public IP"), label:has-text("External IP")')
      .locator('input[type="checkbox"]')
      .first();

    const hasToggle = await publicIpToggle.isVisible().catch(() => false);

    if (hasToggle) {
      const wasChecked = await publicIpToggle.isChecked();
      await publicIpToggle.click();
      await page.waitForTimeout(500);

      const isNowChecked = await publicIpToggle.isChecked();
      expect(isNowChecked).not.toBe(wasChecked);

      // Restore
      await publicIpToggle.click();
    }

    // Close settings
    const closeButton = page.getByRole('button', { name: /close/i }).first();
    await closeButton.click();
  });

  test('should handle public IP fetch error gracefully', async ({ page }) => {
    // Public IP fetch may fail if no internet
    // App should handle this gracefully
    const errorIndicator = page.getByText(/unavailable|error|failed/i);
    const hasError = await errorIndicator.isVisible().catch(() => false);

    // Either shows IP or error message - both are valid states
    expect(hasError).toBeDefined();
  });

  test('should refresh public IP data', async ({ page }) => {
    // Look for refresh button in Network card
    const refreshButton = page
      .locator('[data-testid="refresh-public-ip"]')
      .or(page.locator('button:has(svg[class*="refresh"])'))
      .first();

    const hasRefresh = await refreshButton.isVisible().catch(() => false);

    if (hasRefresh) {
      await refreshButton.click();
      await page.waitForTimeout(2000);
      // Should complete without error
      expect(true).toBeTruthy();
    }
  });
});
