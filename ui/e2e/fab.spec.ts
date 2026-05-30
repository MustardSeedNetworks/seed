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

  // Deleted: "should not trigger tests if FAB is clicked while already
  //   running" — tested window.runAllTestsCount which is never set;
  //   finalCount was always 0 and `expect(0).toBeLessThanOrEqual(1)`
  //   is tautologically true. The disabled-state on click is already
  //   covered by `toBeDisabled()` in the spinner test above.
  // Deleted: "should respect FAB options from settings" — body wrapped
  //   in `if (hasFabSettings)`; silently passes when the panel doesn't
  //   exist (which is always, currently). Tested nothing.
  // Deleted: "should trigger network discovery scan when FAB is
  //   clicked" — final assertion `expect(scanTriggered).toBeDefined()`
  //   on a `let scanTriggered = false` is tautological.
  // See msn-docs-internal/05-Engineering/SEED_E2E_PER_TEST_EVAL_2026-05-26.md
  // for the full evaluation.

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
