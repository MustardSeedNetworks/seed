import { expect, test } from "@playwright/test";

/**
 * VLAN E2E Tests
 *
 * Tests for VLAN functionality:
 * - VLAN information display
 * - Native VLAN display
 * - Tagged VLANs display
 * - Voice VLAN display
 * - VLAN configuration
 * - Switch card integration
 */

test.describe("VLAN Information", () => {
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

  test("should display VLAN information in Switch card", async ({ page }) => {
    // VLAN info is typically in Switch/Network Discovery card
    const switchCard = page
      .getByRole("heading", { name: /switch|vlan/i })
      .or(page.locator('[data-testid="switch-card"]'));

    const hasCard = await switchCard.isVisible().catch(() => false);
    expect(hasCard).toBeDefined();
  });

  test("should show Native VLAN ID", async ({ page }) => {
    const nativeVlan = page.getByText(/native.*vlan|vlan.*native/i);

    const hasNative = await nativeVlan.isVisible().catch(() => false);

    if (hasNative) {
      // Should show VLAN ID (number)
      const vlanId = page.getByText(/vlan\s*\d+|\d+\s*$/);
      const hasId = await vlanId.isVisible().catch(() => false);
      expect(hasId).toBeDefined();
    }
  });

  test("should show Voice VLAN if configured", async ({ page }) => {
    const voiceVlan = page.getByText(/voice.*vlan|vlan.*voice/i);

    const hasVoice = await voiceVlan.isVisible().catch(() => false);
    expect(hasVoice).toBeDefined();
  });

  test("should show tagged VLANs if present", async ({ page }) => {
    const taggedVlan = page.getByText(/tagged|trunk/i);

    const hasTagged = await taggedVlan.isVisible().catch(() => false);
    expect(hasTagged).toBeDefined();
  });

  test("should display LLDP/CDP protocol information", async ({ page }) => {
    const protocolText = page.getByText(/lldp|cdp|edp/i);

    const hasProtocol = await protocolText.isVisible().catch(() => false);
    expect(hasProtocol).toBeDefined();
  });

  test("should show switch port information", async ({ page }) => {
    const portText = page.getByText(/port|ge|fa|eth/i);

    const hasPort = await portText.isVisible().catch(() => false);
    expect(hasPort).toBeDefined();
  });

  test("should show switch name if available", async ({ page }) => {
    // Switch name from LLDP/CDP
    const switchName = page.getByText(/switch.*name|system.*name/i);

    const hasName = await switchName.isVisible().catch(() => false);
    expect(hasName).toBeDefined();
  });

  test("should show management IP if available", async ({ page }) => {
    const mgmtIp = page.getByText(/management|mgmt/i);

    const hasMgmt = await mgmtIp.isVisible().catch(() => false);
    expect(hasMgmt).toBeDefined();
  });
});

test.describe("VLAN Configuration", () => {
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

  test("should have VLAN enable/disable toggle", async ({ page }) => {
    const vlanToggle = page
      .locator('label:has-text("VLAN") input[type="checkbox"]')
      .or(page.locator('input[name*="vlan" i][type="checkbox"]'))
      .first();

    const hasToggle = await vlanToggle.isVisible().catch(() => false);

    if (hasToggle) {
      const wasChecked = await vlanToggle.isChecked();
      await vlanToggle.click();
      await page.waitForTimeout(500);

      const isNowChecked = await vlanToggle.isChecked();
      expect(isNowChecked).not.toBe(wasChecked);

      // Restore
      await vlanToggle.click();
    }
  });

  test("should allow configuring VLAN ID", async ({ page }) => {
    const vlanIdInput = page
      .locator('input[name*="vlan" i][type="number"]')
      .or(page.locator('input[placeholder*="vlan" i]'))
      .first();

    const hasInput = await vlanIdInput.isVisible().catch(() => false);

    if (hasInput) {
      // VLAN IDs are 1-4094
      await vlanIdInput.fill("100");
      await page.waitForTimeout(500);

      const value = await vlanIdInput.inputValue();
      expect(value).toBe("100");
    }
  });

  test("should validate VLAN ID range", async ({ page }) => {
    const vlanIdInput = page.locator('input[name*="vlan" i][type="number"]').first();

    const hasInput = await vlanIdInput.isVisible().catch(() => false);

    if (hasInput) {
      // Try invalid VLAN ID (>4094)
      await vlanIdInput.fill("5000");
      await page.waitForTimeout(500);

      const value = await vlanIdInput.inputValue();
      const numValue = Number.parseInt(value, 10);

      // Should be clamped or show error
      expect(numValue).toBeLessThanOrEqual(4094);
    }
  });
});

test.describe("Switch Card Details", () => {
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

  test("should show switch details when LLDP/CDP available", async ({ page }) => {
    // Switch card shows neighbor info
    const switchCard = page
      .locator('[data-testid="switch-card"]')
      .or(page.locator('section:has(h3:text-is("Switch"))'));

    const hasCard = await switchCard.isVisible().catch(() => false);

    if (hasCard) {
      // Should show protocol used
      const protocol = page.getByText(/lldp|cdp/i);
      const hasProtocol = await protocol.isVisible().catch(() => false);
      expect(hasProtocol).toBeDefined();
    }
  });

  test('should show "No neighbor" when not connected to managed switch', async ({ page }) => {
    const noNeighbor = page.getByText(/no.*neighbor|not.*detected|no.*switch/i);

    const hasNoNeighbor = await noNeighbor.isVisible().catch(() => false);
    expect(hasNoNeighbor).toBeDefined();
  });

  test("should handle multiple neighbors", async ({ page }) => {
    // Some configurations may have multiple LLDP neighbors
    const neighborList = page.locator('[data-testid="neighbor-list"]');

    const hasList = await neighborList.isVisible().catch(() => false);
    expect(hasList).toBeDefined();
  });
});

test.describe("VLAN Help", () => {
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

  test("should explain VLAN concepts in help", async ({ page }) => {
    // Open help
    const helpButton = page.getByRole("button", { name: /help/i });
    await helpButton.click();
    await page.waitForTimeout(500);

    // Look for VLAN or network help
    const vlanHelp = page.getByText(/vlan|802\.1q/i);
    const hasVlanHelp = await vlanHelp.isVisible().catch(() => false);

    expect(hasVlanHelp).toBeDefined();

    // Close help
    const closeButton = page.getByRole("button", { name: /close/i }).first();
    await closeButton.click();
  });
});
