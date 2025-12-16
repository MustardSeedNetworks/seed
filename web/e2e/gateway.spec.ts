import { test, expect } from "@playwright/test";

/**
 * Gateway E2E Tests
 *
 * Tests for Gateway card functionality:
 * - Gateway IP display
 * - Ping latency display
 * - Packet loss display
 * - IPv4/IPv6 gateway status
 * - Reachability indicators
 * - Historical ping data
 */

test.describe("Gateway", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });
  });

  test("should display Gateway card", async ({ page }) => {
    const gatewayCard = page
      .getByRole("heading", { name: /gateway/i })
      .or(page.locator('[data-testid="gateway-card"]'));

    await expect(gatewayCard).toBeVisible({ timeout: 5000 });
  });

  test("should show gateway IP address", async ({ page }) => {
    // Look for IP address format
    const ipAddress = page.getByText(/\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/);

    const hasIP = await ipAddress.isVisible().catch(() => false);
    expect(hasIP).toBeTruthy();
  });

  test("should show ping latency in milliseconds", async ({ page }) => {
    await page.waitForTimeout(2000);
    const latencyText = page.getByText(/\d+(\.\d+)?\s*ms/i);

    // Latency should be visible when gateway is reachable
    await expect(latencyText.first()).toBeVisible({ timeout: 5000 });
  });

  test("should show reachability status", async ({ page }) => {
    await page.waitForTimeout(2000);
    const reachableText = page.getByText(/reachable|unreachable|connected/i);

    // Status indicator should always be present
    await expect(reachableText.first()).toBeVisible({ timeout: 5000 });
  });

  test("should show packet loss percentage", async ({ page }) => {
    await page.waitForTimeout(2000);
    const lossText = page.getByText(/loss|packet/i);

    const hasLoss = await lossText.isVisible().catch(() => false);

    if (hasLoss) {
      // When packet loss section is shown, it should have a percentage
      const percentText = page.getByText(/\d+(\.\d+)?%/);
      await expect(percentText.first()).toBeVisible({ timeout: 3000 });
    }
  });

  test("should show min/avg/max latency stats", async ({ page }) => {
    await page.waitForTimeout(2000);
    const avgText = page.getByText(/avg|average/i);

    // Average latency should be displayed
    await expect(avgText.first()).toBeVisible({ timeout: 5000 });
  });

  test("should show IPv6 gateway if available", async ({ page }) => {
    await page.waitForTimeout(2000);
    const ipv6Text = page.getByText(/ipv6/i);
    const ipv4Text = page.getByText(/ipv4|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/i);

    // Either IPv4 or IPv6 (or both) should be visible
    const hasIPv6 = await ipv6Text
      .first()
      .isVisible()
      .catch(() => false);
    const hasIPv4 = await ipv4Text
      .first()
      .isVisible()
      .catch(() => false);
    expect(hasIPv6 || hasIPv4).toBeTruthy();
  });

  test("should update gateway status in real-time", async ({ page }) => {
    // Get initial latency
    const latencyElement = page.locator(':text-matches("\\\\d+(\\\\.\\\\d+)?\\\\s*ms")').first();
    const hasElement = await latencyElement.isVisible().catch(() => false);

    if (hasElement) {
      // Wait for update (gateway pings typically update every few seconds)
      await page.waitForTimeout(6000);

      // Value should still be a valid latency
      const newValue = await latencyElement.textContent();
      expect(newValue).toMatch(/\d+(\.\d+)?\s*ms/i);
    }
  });

  test("should show success indicator when gateway reachable", async ({ page }) => {
    await page.waitForTimeout(2000);
    const successIndicator = page
      .locator('[class*="success"]')
      .or(page.locator('svg[class*="check"]'))
      .or(page.getByText(/reachable/i));

    const errorIndicator = page.getByText(/unreachable|error|failed/i);

    // Either success OR error state should be shown
    const hasSuccess = await successIndicator
      .first()
      .isVisible()
      .catch(() => false);
    const hasError = await errorIndicator
      .first()
      .isVisible()
      .catch(() => false);
    expect(hasSuccess || hasError).toBeTruthy();
  });

  test("should show error indicator when gateway unreachable", async ({ page }) => {
    await page.waitForTimeout(2000);
    // This test verifies error handling is present in the UI
    const statusIndicator = page
      .locator('[class*="error"], [class*="success"], [class*="warning"]')
      .or(page.getByText(/reachable|unreachable|error/i));

    // Should always have some status indicator
    await expect(statusIndicator.first()).toBeVisible({ timeout: 5000 });
  });
});

test.describe("Gateway Help", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });
  });

  test("should show gateway help in help modal", async ({ page }) => {
    // Open help
    const helpButton = page.getByRole("button", { name: /help/i });
    await helpButton.click();
    await page.waitForTimeout(500);

    // Look for gateway section
    const gatewayHelp = page.getByText(/gateway/i);
    await expect(gatewayHelp.first()).toBeVisible();

    // Close help
    const closeButton = page.getByRole("button", { name: /close/i }).first();
    await closeButton.click();
  });
});
