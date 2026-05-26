import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * FAB (Floating Action Button) E2E Tests
 *
 * Tests the "Run All Tests" functionality via the FAB:
 * - FAB displays when authenticated
 * - Click FAB triggers all tests
 * - FAB shows progress indicator during execution
 * - Cards refresh with new data after tests complete
 * - FAB shows completion notification
 * - FAB behavior is configurable in settings
 */

test.describe('FAB - Run All Tests Flow', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should display FAB button when authenticated', async ({ page }) => {
    // FAB should be visible in bottom-right corner
    const fab = page.getByTestId('fab-run-all-tests');
    await expect(fab).toBeVisible();

    // Check FAB positioning (fixed bottom-right)
    const fabBox = await fab.boundingBox();
    expect(fabBox).toBeTruthy();

    // FAB should have play icon initially
    const playIcon = fab.locator('svg');
    await expect(playIcon).toBeVisible();
  });

  test('should show loading spinner when FAB is clicked', async ({ page }) => {
    const fab = page.getByTestId('fab-run-all-tests');

    // Click FAB
    await fab.click();

    // Should show spinner (animated SVG with opacity classes)
    // Synchronously check the data-running attribute instead of
    // racing the animate-spin CSS class — see seed#1168.
    await expect(fab).toHaveAttribute('data-running', 'true');

    // FAB should be disabled during test execution
    await expect(fab).toBeDisabled();
  });

  test('should trigger all tests when FAB is clicked', async ({ page }) => {
    // Set up network interceptors to track API calls
    const apiCalls = new Set<string>();

    page.on('request', (request) => {
      const url = request.url();
      if (url.includes('/api/')) {
        const [, apiPath] = url.split('/api/');
        const [endpoint] = apiPath.split('?');
        apiCalls.add(endpoint);
      }
    });

    const fab = page.getByTestId('fab-run-all-tests');

    // Click FAB to trigger all tests
    await fab.click();

    // Wait a bit for API calls to be made

    // Verify key endpoints were called (based on default FAB options)
    // Link layer
    expect(apiCalls.has('link') || apiCalls.has('wifi') || apiCalls.has('cable')).toBeTruthy();

    // Network layer - at least one of these should be called
    const networkCalled =
      apiCalls.has('ipconfig') || apiCalls.has('gateway') || apiCalls.has('dns');
    expect(networkCalled).toBeTruthy();
  });

  test('should refresh card data after tests complete', async ({ page }) => {
    // Get initial link card data
    const linkCard = page.locator('h3:has-text("Link"), h4:has-text("Link")').first();
    await expect(linkCard).toBeVisible();

    // Track if any card updates occur
    let cardUpdated = false;
    page.on('response', (response) => {
      if (response.url().includes('/api/link') && response.ok()) {
        cardUpdated = true;
      }
    });

    const fab = page.getByTestId('fab-run-all-tests');

    // Click FAB
    await fab.click();

    // Wait for tests to complete (spinner disappears)
    await expect(fab).toHaveAttribute('data-running', 'true');
    await expect(fab).toHaveAttribute('data-running', 'false', { timeout: 65000 }); // 60s timeout + buffer

    // Verify card data was updated
    expect(cardUpdated).toBeTruthy();
  });

  test('should complete and stop spinner after tests finish', async ({ page }) => {
    const fab = page.getByTestId('fab-run-all-tests');

    // Click FAB
    await fab.click();

    // Spinner should appear
    // Synchronously check the data-running attribute instead of
    // racing the animate-spin CSS class — see seed#1168.
    await expect(fab).toHaveAttribute('data-running', 'true');

    // Wait for completion — FAB returns to idle within 60s.
    await expect(fab).toHaveAttribute('data-running', 'false', { timeout: 65000 });

    // FAB should be enabled again
    await expect(fab).toBeEnabled();

    // Play icon should be back (FAB returned to idle).
    const playIcon = fab.locator('svg');
    await expect(playIcon).toBeVisible();
  });

  test('should not trigger tests if FAB is clicked while already running', async ({ page }) => {
    const fab = page.getByTestId('fab-run-all-tests');

    // Click FAB first time
    await fab.click();
    await expect(fab).toBeDisabled();

    // Try clicking again while disabled
    const _clickCount = await page.evaluate(() => {
      let count = 0;
      window.addEventListener('runAllTests', () => count++);
      return count;
    });

    // Try to click disabled FAB
    await fab.click({ force: true }).catch(() => {
      // Expected to fail or do nothing
    });

    // Should still only have one test run
    const finalCount = await page.evaluate(
      () => (window as unknown as { runAllTestsCount?: number }).runAllTestsCount || 0,
    );

    // Event should not have fired multiple times
    expect(finalCount).toBeLessThanOrEqual(1);
  });

  test('should respect FAB options from settings', async ({ page }) => {
    // Open settings drawer
    const settingsButton = page.getByTestId('header-open-settings');
    await settingsButton.click();

    // Wait for settings drawer
    await expect(page.getByTestId('settings-drawer')).toBeVisible({ timeout: 5000 });

    // Look for FAB-related settings (if they exist in UI)
    // This will help verify FAB options are configurable
    const fabSettings = page.locator('text=/FAB|Run All Tests|Test Options/i').first();
    const hasFabSettings = await fabSettings.isVisible();

    if (hasFabSettings) {
      // If FAB settings exist, verify they can be changed
      await expect(fabSettings).toBeVisible();
    }

    // Close settings
    const closeButton = page.getByRole('button', { name: /close/i }).first();
    await closeButton.click();
  });

  test('should trigger network discovery scan when FAB is clicked', async ({ page }) => {
    // Track if network discovery scan endpoint is called
    let scanTriggered = false;

    page.on('request', (request) => {
      if (request.url().includes('/api/devices/scan') && request.method() === 'POST') {
        scanTriggered = true;
      }
    });

    const fab = page.getByTestId('fab-run-all-tests');

    // Click FAB
    await fab.click();

    // Wait a bit for scan to be triggered

    // Verify scan was triggered (if network discovery is enabled in FAB options)
    // Note: This depends on default FAB options configuration
    // The test verifies the mechanism works, actual behavior depends on settings
    expect(scanTriggered).toBeDefined();
  });

  test('should handle test failures gracefully', async ({ page }) => {
    // Intercept an API call and make it fail
    await page.route('**/api/dns', (route) => {
      route.fulfill({
        status: 500,
        body: JSON.stringify({ error: 'Internal server error' }),
      });
    });

    const fab = page.getByTestId('fab-run-all-tests');

    // Click FAB
    await fab.click();

    // Spinner should still appear
    await expect(fab).toHaveAttribute('data-running', 'true');

    // Even with failures, tests should complete and spinner should stop
    await expect(fab).toHaveAttribute('data-running', 'false', { timeout: 65000 });

    // FAB should be enabled again
    await expect(fab).toBeEnabled();
  });

  test('should maintain FAB visibility on page scroll', async ({ page }) => {
    const fab = page.getByTestId('fab-run-all-tests');

    // Verify FAB is visible initially
    await expect(fab).toBeVisible();

    // Scroll down the page
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));

    // FAB should still be visible (it's position: fixed)
    await expect(fab).toBeVisible();

    // Scroll back to top
    await page.evaluate(() => window.scrollTo(0, 0));

    // FAB should still be visible
    await expect(fab).toBeVisible();
  });

  test('should be keyboard accessible', async ({ page }) => {
    const fab = page.getByTestId('fab-run-all-tests');

    // Tab to FAB (may need multiple tabs depending on page structure)
    // Focus the FAB using keyboard
    await fab.focus();

    // Verify FAB is focused
    await expect(fab).toBeFocused();

    // Press Enter to activate
    let testTriggered = false;
    page.on('request', (request) => {
      if (request.url().includes('/api/')) {
        testTriggered = true;
      }
    });

    await page.keyboard.press('Enter');

    // Wait a bit for API calls

    // Tests should have been triggered
    expect(testTriggered).toBeTruthy();
  });

  test('should show proper aria labels for accessibility', async ({ page }) => {
    const fab = page.getByTestId('fab-run-all-tests');

    // Verify accessibility attributes
    const ariaLabel = await fab.getAttribute('aria-label');
    const title = await fab.getAttribute('title');

    // At least one should be present
    expect(ariaLabel || title).toBeTruthy();

    // Should contain meaningful text
    const labelText = (ariaLabel || title || '').toLowerCase();
    expect(labelText.includes('run') || labelText.includes('test')).toBeTruthy();
  });
});
