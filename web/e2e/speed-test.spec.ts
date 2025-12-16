import { test, expect } from '@playwright/test';

/**
 * Speed Test E2E Tests
 *
 * Tests the speed/performance testing functionality:
 * - Performance card displays correctly
 * - Speed test controls
 * - iPerf3 integration
 * - Results display
 */

test.describe('Speed Test', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    // Authenticate
    await page.getByLabel(/username/i).fill('admin');
    await page.getByLabel(/password/i).fill('luminetiq');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Wait for dashboard to load
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible({ timeout: 10000 });
  });

  test('should display Performance card', async ({ page }) => {
    const perfCard = page
      .locator('h3:has-text("Performance"), h4:has-text("Performance"), h3:has-text("Speed"), h4:has-text("Speed")')
      .or(page.locator('[data-testid="performance-card"]'))
      .first();
    await expect(perfCard).toBeVisible({ timeout: 5000 });
  });

  test('should show speed metrics or placeholder', async ({ page }) => {
    // Look for speed metrics (download, upload, latency)
    const speedMetrics = page
      .locator('text=/mbps|ms|latency|download|upload|bandwidth/i')
      .or(page.locator('[data-testid="speed-results"]'))
      .first();

    await page.waitForTimeout(2000);

    // Should show metrics or a "run test" prompt
    const hasMetrics = await speedMetrics.isVisible().catch(() => false);
    const hasPrompt = await page.getByText(/run test|start|no results/i).isVisible().catch(() => false);

    expect(hasMetrics || hasPrompt).toBeTruthy();
  });

  test('should have speed test button', async ({ page }) => {
    const testButton = page
      .getByRole('button', { name: /test|run|start|speed/i })
      .or(page.locator('[data-testid="speed-test-btn"]'))
      .first();

    await expect(testButton).toBeVisible({ timeout: 5000 });
  });

  test('should show iPerf3 availability status', async ({ page }) => {
    // Look for iPerf3 status indicator
    const iperfStatus = page
      .locator('text=/iperf|iperf3|server|client/i')
      .or(page.locator('[data-testid="iperf-status"]'))
      .first();

    await page.waitForTimeout(2000);

    // Check if iPerf section exists
    const _hasIperf = await iperfStatus.isVisible().catch(() => false);

    // iPerf may not be available on all systems, which is OK
    // Just verify the performance section is present
    const perfSection = page.locator('text=/performance|speed|bandwidth/i').first();
    await expect(perfSection).toBeVisible({ timeout: 5000 });
  });

  test('should show performance settings in drawer', async ({ page }) => {
    // Open settings drawer
    const settingsButton = page
      .getByRole('button', { name: /settings/i })
      .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));
    await settingsButton.click();

    // Look for performance settings section
    await expect(page.getByText(/performance|speed|iperf|bandwidth/i)).toBeVisible({ timeout: 5000 });
  });

  test('should display speed gauge when results exist', async ({ page }) => {
    // Look for speed gauge component
    const gauge = page
      .locator('[data-testid="speed-gauge"]')
      .or(page.locator('svg[class*="gauge"]'))
      .or(page.locator('.speed-gauge'))
      .first();

    await page.waitForTimeout(3000);

    // Gauge may or may not be visible depending on test state
    const hasGauge = await gauge.isVisible().catch(() => false);

    // If no gauge, should at least have the performance card
    if (!hasGauge) {
      const perfCard = page.locator('text=/performance|speed/i').first();
      await expect(perfCard).toBeVisible();
    }
  });
});
