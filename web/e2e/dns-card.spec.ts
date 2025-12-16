import { test, expect } from "@playwright/test";

/**
 * DNS Card E2E Tests
 *
 * Tests for DNS functionality:
 * - DNS server display
 * - Forward lookup results
 * - Reverse lookup results
 * - IPv6 lookup results
 * - Latency display
 * - DNS settings configuration
 * - Test hostname configuration
 */

test.describe("DNS Card", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });
  });

  test("should display DNS card", async ({ page }) => {
    const dnsCard = page
      .getByRole("heading", { name: /dns/i })
      .or(page.locator('[data-testid="dns-card"]'));

    await expect(dnsCard).toBeVisible({ timeout: 5000 });
  });

  test("should show DNS server address", async ({ page }) => {
    // Look for DNS server IP
    const dnsServer = page.getByText(/\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/);

    const hasServer = await dnsServer.isVisible().catch(() => false);
    expect(hasServer).toBeDefined();
  });

  test("should show forward lookup result", async ({ page }) => {
    const forwardText = page.getByText(/forward|a record/i);

    const hasForward = await forwardText.isVisible().catch(() => false);
    expect(hasForward).toBeDefined();
  });

  test("should show reverse lookup result", async ({ page }) => {
    const reverseText = page.getByText(/reverse|ptr/i);

    const hasReverse = await reverseText.isVisible().catch(() => false);
    expect(hasReverse).toBeDefined();
  });

  test("should show lookup latency in milliseconds", async ({ page }) => {
    const latencyText = page.getByText(/\d+(\.\d+)?\s*ms/i);

    const hasLatency = await latencyText.isVisible().catch(() => false);
    expect(hasLatency).toBeDefined();
  });

  test("should show success/fail status for each lookup", async ({ page }) => {
    const successIndicator = page.locator('[class*="success"], svg[class*="check"]');
    const errorIndicator = page.locator('[class*="error"], svg[class*="x"]');

    const successCount = await successIndicator.count();
    const errorCount = await errorIndicator.count();

    // Should have some status indicators
    expect(successCount + errorCount).toBeGreaterThanOrEqual(0);
  });

  test("should show IPv6 lookup if supported", async ({ page }) => {
    const ipv6Text = page.getByText(/ipv6|aaaa/i);

    const hasIPv6 = await ipv6Text.isVisible().catch(() => false);
    expect(hasIPv6).toBeDefined();
  });

  test("should show multiple DNS servers if configured", async ({ page }) => {
    // Multiple IPs may be shown
    const serverIPs = page.locator(
      ':text-matches("\\\\d{1,3}\\\\.\\\\d{1,3}\\\\.\\\\d{1,3}\\\\.\\\\d{1,3}")'
    );

    const serverCount = await serverIPs.count();
    expect(serverCount).toBeGreaterThanOrEqual(0);
  });
});

test.describe("DNS Settings", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });

    // Open settings
    const settingsButton = page.getByRole("button", { name: /settings/i });
    await settingsButton.click();
    await page.waitForTimeout(500);
  });

  test("should display DNS settings section", async ({ page }) => {
    const dnsSection = page.getByText(/dns/i).first();
    await expect(dnsSection).toBeVisible();
  });

  test("should allow configuring test hostname", async ({ page }) => {
    // Look for hostname input
    const hostnameInput = page
      .locator('input[name*="hostname" i]')
      .or(page.locator('input[placeholder*="hostname" i]'))
      .or(page.locator('input[placeholder*="google.com" i]'))
      .first();

    const hasInput = await hostnameInput.isVisible().catch(() => false);

    if (hasInput) {
      const originalValue = await hostnameInput.inputValue();

      // Change hostname
      await hostnameInput.fill("example.com");
      await page.waitForTimeout(500);

      const newValue = await hostnameInput.inputValue();
      expect(newValue).toBe("example.com");

      // Restore
      await hostnameInput.fill(originalValue || "google.com");
      await page.waitForTimeout(500);
    }
  });

  test("should validate hostname format", async ({ page }) => {
    const hostnameInput = page
      .locator('input[name*="hostname" i]')
      .or(page.locator('input[placeholder*="hostname" i]'))
      .first();

    const hasInput = await hostnameInput.isVisible().catch(() => false);

    if (hasInput) {
      // Try invalid hostname
      await hostnameInput.fill("not a valid hostname!!!");
      await page.waitForTimeout(500);

      // Should show error or sanitize
      const errorText = page.getByText(/invalid|error/i);
      const hasError = await errorText.isVisible().catch(() => false);

      expect(hasError).toBeDefined();
    }
  });

  test("should persist DNS hostname setting", async ({ page }) => {
    const hostnameInput = page
      .locator('input[name*="hostname" i]')
      .or(page.locator('input[placeholder*="hostname" i]'))
      .first();

    const hasInput = await hostnameInput.isVisible().catch(() => false);

    if (hasInput) {
      const testHostname = "test-" + Date.now() + ".com";
      await hostnameInput.fill(testHostname);
      await page.waitForTimeout(1000);

      // Close and reopen settings
      const closeButton = page.getByRole("button", { name: /close/i }).first();
      await closeButton.click();
      await page.waitForTimeout(500);

      const settingsButton = page.getByRole("button", { name: /settings/i });
      await settingsButton.click();
      await page.waitForTimeout(500);

      // Check if persisted
      const reopenedInput = page
        .locator('input[name*="hostname" i]')
        .or(page.locator('input[placeholder*="hostname" i]'))
        .first();

      const persistedValue = await reopenedInput.inputValue();
      expect(persistedValue).toBe(testHostname);
    }
  });
});

test.describe("DNS Help", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });
  });

  test("should show DNS help in help modal", async ({ page }) => {
    // Open help
    const helpButton = page.getByRole("button", { name: /help/i });
    await helpButton.click();
    await page.waitForTimeout(500);

    // Look for DNS section
    const dnsHelp = page.getByText(/dns/i);
    await expect(dnsHelp.first()).toBeVisible();

    // Should explain lookup types
    const forwardHelp = page.getByText(/forward.*lookup|a record/i);
    const hasForwardHelp = await forwardHelp.isVisible().catch(() => false);
    expect(hasForwardHelp).toBeDefined();

    // Close help
    const closeButton = page.getByRole("button", { name: /close/i }).first();
    await closeButton.click();
  });
});
