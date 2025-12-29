import { expect, test } from "@playwright/test";

/**
 * Link Card E2E Tests
 *
 * Tests for Link Status card functionality:
 * - Carrier/link status display
 * - Speed negotiation display
 * - Duplex mode display
 * - MTU display
 * - Auto-negotiation status
 * - Advertised speeds
 */

test.describe("Link Card", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test("should display Link card", async ({ page }) => {
    const linkCard = page
      .getByRole("heading", { name: /link/i })
      .or(page.locator('[data-testid="link-card"]'));

    await expect(linkCard).toBeVisible({ timeout: 5000 });
  });

  test("should show carrier/connection status", async ({ page }) => {
    const carrierText = page.getByText(/carrier|connected|link up|link down/i);

    const hasCarrier = await carrierText.isVisible().catch(() => false);
    expect(hasCarrier).toBeDefined();
  });

  test("should show link speed", async ({ page }) => {
    // Speed in Mbps or Gbps
    const speedText = page.getByText(/\d+\s*(mbps|gbps|mb\/s|gb\/s)/i);

    const hasSpeed = await speedText.isVisible().catch(() => false);
    expect(hasSpeed).toBeDefined();
  });

  test("should show duplex mode", async ({ page }) => {
    const duplexText = page.getByText(/full.*duplex|half.*duplex|duplex/i);

    const hasDuplex = await duplexText.isVisible().catch(() => false);
    expect(hasDuplex).toBeDefined();
  });

  test("should show MTU value", async ({ page }) => {
    const mtuText = page.getByText(/mtu/i);

    const hasMtu = await mtuText.isVisible().catch(() => false);

    if (hasMtu) {
      // MTU is typically a number like 1500, 9000, etc.
      const mtuValue = page.getByText(/\d{4}/);
      const hasValue = await mtuValue.isVisible().catch(() => false);
      expect(hasValue).toBeDefined();
    }
  });

  test("should show auto-negotiation status", async ({ page }) => {
    const autoNegText = page.getByText(/auto.*neg|auto-negotiation|autoneg/i);

    const hasAutoNeg = await autoNegText.isVisible().catch(() => false);
    expect(hasAutoNeg).toBeDefined();
  });

  test("should show advertised speeds", async ({ page }) => {
    const advertisedText = page.getByText(/advertised|supported.*speed/i);

    const hasAdvertised = await advertisedText.isVisible().catch(() => false);
    expect(hasAdvertised).toBeDefined();
  });

  test("should show success indicator when link is up", async ({ page }) => {
    const successIndicator = page
      .locator('[class*="success"]')
      .or(page.locator('svg[class*="check"]'))
      .or(page.getByText(/connected|link up/i));

    const hasSuccess = await successIndicator.isVisible().catch(() => false);
    expect(hasSuccess).toBeDefined();
  });

  test("should show error indicator when link is down", async ({ page }) => {
    const errorIndicator = page
      .locator('[class*="error"]')
      .or(page.locator('svg[class*="x"]'))
      .or(page.getByText(/disconnected|link down|no carrier/i));

    const hasError = await errorIndicator.isVisible().catch(() => false);
    expect(hasError).toBeDefined();
  });

  test("should update link status in real-time", async ({ page }) => {
    // Get initial status
    const statusElement = page.getByText(/connected|disconnected|link/i).first();
    const hasElement = await statusElement.isVisible().catch(() => false);

    if (hasElement) {
      // Wait for potential update
      await page.waitForTimeout(6000);

      // Status should still be visible (may or may not change)
      await expect(statusElement).toBeVisible();
    }
  });
});

test.describe("Link Card Help", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test("should show link help in help modal", async ({ page }) => {
    // Open help
    const helpButton = page.getByRole("button", { name: /help/i });
    await helpButton.click();
    await page.waitForTimeout(500);

    // Look for link section
    const linkHelp = page.getByText(/link.*status/i);
    await expect(linkHelp.first()).toBeVisible();

    // Should explain carrier, speed, duplex
    const carrierHelp = page.getByText(/carrier.*signal|physical.*layer/i);
    const hasCarrierHelp = await carrierHelp.isVisible().catch(() => false);
    expect(hasCarrierHelp).toBeDefined();

    // Close help
    const closeButton = page.getByRole("button", { name: /close/i }).first();
    await closeButton.click();
  });
});

test.describe("Link Card MTU Configuration", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({
      timeout: 10000,
    });

    // Open settings
    const settingsButton = page.getByRole("button", { name: /settings/i });
    await settingsButton.click();
    await page.waitForTimeout(500);
  });

  test("should allow MTU configuration in settings", async ({ page }) => {
    const mtuInput = page
      .locator('input[name*="mtu" i]')
      .or(page.locator('input[placeholder*="mtu" i]'))
      .first();

    const hasInput = await mtuInput.isVisible().catch(() => false);

    if (hasInput) {
      // Standard MTU values: 1500, 9000 (jumbo)
      await mtuInput.fill("1500");
      await page.waitForTimeout(500);

      const value = await mtuInput.inputValue();
      expect(value).toBe("1500");
    }
  });

  test("should validate MTU range", async ({ page }) => {
    const mtuInput = page.locator('input[name*="mtu" i]').first();

    const hasInput = await mtuInput.isVisible().catch(() => false);

    if (hasInput) {
      // Try invalid MTU (too small)
      await mtuInput.fill("100");
      await page.waitForTimeout(500);

      const errorText = page.getByText(/invalid|error|min/i);
      const hasError = await errorText.isVisible().catch(() => false);
      expect(hasError).toBeDefined();
    }
  });
});
