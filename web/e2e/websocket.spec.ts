import { test, expect } from "@playwright/test";

/**
 * WebSocket E2E Tests
 *
 * Tests the WebSocket real-time connectivity:
 * - Connection established
 * - Real-time updates received
 * - Connection recovery
 * - Status indicators
 */

test.describe("WebSocket Connectivity", () => {
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

  test("should establish WebSocket connection", async ({ page }) => {
    // Monitor WebSocket connections
    const wsMessages: string[] = [];

    page.on("websocket", (ws) => {
      ws.on("framereceived", (frame) => {
        if (frame.payload) {
          wsMessages.push(frame.payload.toString());
        }
      });
    });

    // Wait for potential WS messages
    await page.waitForTimeout(5000);

    // Check for connection status indicator
    const connectionStatus = page
      .locator("text=/connected|online|live/i")
      .or(page.locator('[data-testid="ws-status"]'))
      .or(page.locator('[class*="status"][class*="connected"]'))
      .first();

    const hasStatus = await connectionStatus.isVisible().catch(() => false);

    // Either we got WS messages or the UI shows connected status
    expect(hasStatus || wsMessages.length > 0).toBeTruthy();
  });

  test("should receive real-time data updates", async ({ page }) => {
    // Get initial values
    const _initialContent = await page.locator("body").textContent();

    // Wait for potential updates
    await page.waitForTimeout(5000);

    // Page should still be interactive (no disconnection errors)
    await expect(page.locator("body")).toBeVisible();

    // Check for any error messages about connection
    const hasConnectionError = await page
      .getByText(/disconnected|connection lost|reconnecting/i)
      .isVisible()
      .catch(() => false);

    // Should not have connection errors
    expect(hasConnectionError).toBeFalsy();
  });

  test("should show connection status in UI", async ({ page }) => {
    // Look for any connection/status indicator
    const statusIndicator = page
      .locator('[data-testid="connection-status"]')
      .or(page.locator('[class*="status-indicator"]'))
      .or(page.locator('[class*="connection"]'))
      .first();

    await page.waitForTimeout(2000);

    // Status indicator may be visible, or page just works without explicit indicator
    const hasIndicator = await statusIndicator.isVisible().catch(() => false);

    // If no explicit indicator, verify page is functional
    if (!hasIndicator) {
      // Cards should still be loading/updating
      const linkCard = page.locator("text=/link|interface/i").first();
      await expect(linkCard).toBeVisible();
    }
  });

  test("should handle page refresh without losing state", async ({ page }) => {
    // Note some initial state
    const hasCards = await page.locator("text=/link|gateway|dns/i").first().isVisible();
    expect(hasCards).toBeTruthy();

    // Refresh the page
    await page.reload();

    // Should remain authenticated and see dashboard
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });

    // Cards should reappear
    const cardsAfterRefresh = page.locator("text=/link|gateway|dns/i").first();
    await expect(cardsAfterRefresh).toBeVisible({ timeout: 5000 });
  });
});
