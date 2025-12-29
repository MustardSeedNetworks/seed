import { expect, test } from "@playwright/test";

/**
 * Network Discovery E2E Tests
 *
 * Tests the network discovery functionality:
 * - Discovery card displays correctly
 * - Device list renders
 * - Scan controls work
 * - Device details modal
 * - Filtering and sorting
 */

test.describe("Network Discovery", () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    // Authenticate
    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();

    // Wait for dashboard to load
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test("should display Network Discovery card", async ({ page }) => {
    const discoveryCard = page
      .locator('h3:has-text("Discovery"), h4:has-text("Discovery")')
      .or(page.locator('[data-testid="network-discovery-card"]'))
      .first();
    await expect(discoveryCard).toBeVisible({ timeout: 5000 });
  });

  test("should show discovered devices count", async ({ page }) => {
    // Look for device count indicator
    const deviceCount = page
      .locator("text=/\\d+\\s*(devices?|hosts?)/i")
      .or(page.locator('[data-testid="device-count"]'))
      .first();

    // Should show some indication of devices (even if 0)
    await expect(deviceCount).toBeVisible({ timeout: 10000 });
  });

  test("should have scan button available", async ({ page }) => {
    const scanButton = page
      .getByRole("button", { name: /scan|discover|refresh/i })
      .or(page.locator('button:has(svg[class*="refresh"], svg[class*="scan"])'))
      .first();

    await expect(scanButton).toBeVisible({ timeout: 5000 });
  });

  test("should show discovery settings in drawer", async ({ page }) => {
    // Open settings drawer
    const settingsButton = page
      .getByRole("button", { name: /settings/i })
      .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));
    await settingsButton.click();

    // Look for discovery settings section
    await expect(page.getByText(/discovery/i)).toBeVisible({ timeout: 5000 });
  });

  test("should display device list when devices exist", async ({ page }) => {
    // Wait for any device list to appear (table or list)
    const deviceList = page
      .locator('table, [role="table"], [data-testid="device-list"]')
      .or(page.locator('.device-item, [data-testid="device-row"]'))
      .first();

    // Give time for discovery to complete - may be empty initially
    await page.waitForTimeout(2000);

    // Check if there's either a device list or a "no devices" message
    const hasDevices = await deviceList.isVisible().catch(() => false);
    const hasNoDevicesMsg = await page
      .getByText(/no devices|no hosts|empty/i)
      .isVisible()
      .catch(() => false);

    expect(hasDevices || hasNoDevicesMsg).toBeTruthy();
  });
});
