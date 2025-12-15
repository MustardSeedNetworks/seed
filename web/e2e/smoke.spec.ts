import { test, expect } from '@playwright/test';

/**
 * Smoke Tests
 *
 * Quick sanity checks to verify the app is running:
 * - Page loads without errors
 * - No console errors
 * - Basic UI elements render
 */

test.describe('Smoke Tests', () => {
  test('should load the application without errors', async ({ page }) => {
    const errors: string[] = [];

    // Capture console errors
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.goto('/');

    // Page should have loaded something
    await expect(page.locator('body')).not.toBeEmpty();

    // Filter out expected errors (like 401 when not authenticated)
    const criticalErrors = errors.filter(
      (e) => !e.includes('401') && !e.includes('Unauthorized') && !e.includes('Failed to fetch')
    );

    // No critical console errors
    expect(criticalErrors).toHaveLength(0);
  });

  test('should have proper page title', async ({ page }) => {
    await page.goto('/');

    // Title should contain app name
    await expect(page).toHaveTitle(/luminetiq|netscope|network/i);
  });

  test('should have proper viewport and be responsive', async ({ page }) => {
    await page.goto('/');

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Content should still be visible
    await expect(page.locator('body')).toBeVisible();

    // Set desktop viewport
    await page.setViewportSize({ width: 1920, height: 1080 });

    // Content should still be visible
    await expect(page.locator('body')).toBeVisible();
  });

  test('should handle 404 routes gracefully', async ({ page }) => {
    await page.goto('/nonexistent-route-12345');

    // Should either redirect to login or show 404 page
    const hasContent = await page.locator('body').textContent();
    expect(hasContent).toBeTruthy();
  });
});
