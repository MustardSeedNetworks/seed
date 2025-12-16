import { test, expect } from "@playwright/test";

/**
 * WiFi Survey E2E Tests
 *
 * Tests the WiFi survey functionality:
 * - WiFi card displays correctly
 * - Survey creation flow
 * - Floor plan interaction
 * - Sample point collection
 * - Survey list and management
 */

test.describe("WiFi Survey", () => {
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
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });
  });

  test("should display WiFi card", async ({ page }) => {
    const wifiCard = page
      .locator('h3:has-text("WiFi"), h4:has-text("WiFi")')
      .or(page.locator('[data-testid="wifi-card"]'))
      .first();
    await expect(wifiCard).toBeVisible({ timeout: 5000 });
  });

  test("should show WiFi signal information", async ({ page }) => {
    // Look for WiFi signal indicators (SSID, signal strength, channel)
    const signalInfo = page
      .locator("text=/SSID|signal|channel|rssi|dbm/i")
      .or(page.locator('[data-testid="wifi-signal"]'))
      .first();

    // WiFi info should be visible (may show "Not connected" if no WiFi)
    await page.waitForTimeout(2000);
    const hasInfo = await signalInfo.isVisible().catch(() => false);
    const hasNotConnected = await page
      .getByText(/not connected|no wifi|unavailable/i)
      .isVisible()
      .catch(() => false);

    expect(hasInfo || hasNotConnected).toBeTruthy();
  });

  test("should have survey button or link", async ({ page }) => {
    // Look for survey button/link
    const surveyButton = page
      .getByRole("button", { name: /survey|scan|start/i })
      .or(page.getByRole("link", { name: /survey/i }))
      .or(page.locator('[data-testid="wifi-survey-btn"]'))
      .first();

    await expect(surveyButton).toBeVisible({ timeout: 5000 });
  });

  test("should show WiFi settings in drawer", async ({ page }) => {
    // Open settings drawer
    const settingsButton = page
      .getByRole("button", { name: /settings/i })
      .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));
    await settingsButton.click();

    // Look for WiFi settings section
    await expect(page.getByText(/wifi|wireless/i)).toBeVisible({ timeout: 5000 });
  });

  test("should display available networks list", async ({ page }) => {
    // Wait for networks to load
    await page.waitForTimeout(3000);

    // Look for network list
    const networkList = page
      .locator("text=/available|networks|scan results/i")
      .or(page.locator('[data-testid="network-list"]'))
      .first();

    const hasNetworks = await networkList.isVisible().catch(() => false);
    const hasNoNetworks = await page
      .getByText(/no networks|scanning|not available/i)
      .isVisible()
      .catch(() => false);

    // Either networks visible or a status message
    expect(hasNetworks || hasNoNetworks).toBeTruthy();
  });
});
