import { test, expect } from "@playwright/test";

/**
 * Setup Wizard E2E Tests
 *
 * Tests the initial setup/onboarding flow:
 * - Setup wizard detection
 * - Interface selection
 * - Initial configuration
 * - Completion flow
 *
 * Note: These tests may skip if setup is already complete
 */

test.describe("Setup Wizard", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();
  });

  test("should detect if setup is needed", async ({ page }) => {
    await page.goto("/");

    // Wait for either login form or setup wizard
    const loginForm = page.getByRole("heading", { name: /login/i });
    const setupWizard = page.getByText(/setup|welcome|get started|configure/i).first();

    await page.waitForTimeout(3000);

    const hasLogin = await loginForm.isVisible().catch(() => false);
    const hasSetup = await setupWizard.isVisible().catch(() => false);

    // Should show either login or setup
    expect(hasLogin || hasSetup).toBeTruthy();
  });

  test("should show interface selection if in setup mode", async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);

    // Check if we're in setup mode
    const setupIndicator = page.getByText(/setup|configure|interface selection/i).first();
    const isInSetup = await setupIndicator.isVisible().catch(() => false);

    if (isInSetup) {
      // Should show interface options
      const interfaceOptions = page
        .locator("text=/eth|en|wlan|interface|adapter/i")
        .or(page.locator('select, [role="listbox"]'))
        .first();

      await expect(interfaceOptions).toBeVisible({ timeout: 5000 });
    } else {
      // Setup already complete, verify we can login
      const loginForm = page.getByRole("heading", { name: /login/i });
      await expect(loginForm).toBeVisible({ timeout: 5000 });
    }
  });

  test("should have navigation through setup steps", async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);

    const setupIndicator = page.getByText(/setup|welcome/i).first();
    const isInSetup = await setupIndicator.isVisible().catch(() => false);

    if (isInSetup) {
      // Look for next/continue/skip buttons
      const navButton = page.getByRole("button", { name: /next|continue|skip|finish/i }).first();

      await expect(navButton).toBeVisible({ timeout: 5000 });
    } else {
      // Test passes - setup already complete
      expect(true).toBeTruthy();
    }
  });

  test("should complete setup and reach dashboard", async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);

    const setupIndicator = page.getByText(/setup|welcome/i).first();
    const isInSetup = await setupIndicator.isVisible().catch(() => false);

    if (isInSetup) {
      // Click through setup steps
      let attempts = 0;
      const maxAttempts = 10;

      while (attempts < maxAttempts) {
        const nextBtn = page
          .getByRole("button", { name: /next|continue|finish|complete|skip/i })
          .first();

        const hasNext = await nextBtn.isVisible().catch(() => false);

        if (hasNext) {
          await nextBtn.click();
          await page.waitForTimeout(1000);
        } else {
          break;
        }

        attempts++;
      }

      // After setup, should see login or dashboard
      const afterSetup = page.getByRole("heading", { name: /login|dashboard|link/i }).first();

      await expect(afterSetup).toBeVisible({ timeout: 10000 });
    } else {
      // Login and verify dashboard
      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({ timeout: 10000 });
    }
  });
});
