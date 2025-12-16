import { test, expect } from "@playwright/test";

/**
 * Authentication E2E Tests
 *
 * Tests the login flow and token handling:
 * - Login form renders correctly
 * - Login with valid credentials
 * - Login with invalid credentials shows error
 * - Logout clears session
 */

test.describe("Authentication", () => {
  test.beforeEach(async ({ page }) => {
    // Clear any stored tokens
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();
  });

  test("should display login form when not authenticated", async ({ page }) => {
    await page.goto("/");

    // Check for login form elements
    await expect(page.getByRole("heading", { name: /login/i })).toBeVisible();
    await expect(page.getByLabel(/username/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await expect(page.getByRole("button", { name: /sign in|login/i })).toBeVisible();
  });

  test("should show error with invalid credentials", async ({ page }) => {
    await page.goto("/");

    // Fill in invalid credentials
    await page.getByLabel(/username/i).fill("wronguser");
    await page.getByLabel(/password/i).fill("wrongpassword");
    await page.getByRole("button", { name: /sign in|login/i }).click();

    // Should show error message
    await expect(page.getByText(/invalid|incorrect|failed/i)).toBeVisible({ timeout: 5000 });
  });

  test("should login successfully with valid credentials", async ({ page }) => {
    await page.goto("/");

    // Fill in valid credentials (default admin/seed)
    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();

    // Should redirect to dashboard
    await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
      timeout: 10000,
    });
  });
});
