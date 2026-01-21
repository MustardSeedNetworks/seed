import { expect, test } from '@playwright/test';

/**
 * Vulnerability Scanning E2E Tests
 *
 * Tests the vulnerability scanning functionality:
 * - Vulnerability section displays
 * - Scan controls work
 * - Results display
 * - Device vulnerability details
 */

test.describe('Vulnerability Scanning', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    // Authenticate
    await page.getByLabel(/username/i).fill('admin');
    await page.getByLabel(/password/i).fill('seed');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Wait for dashboard to load
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should have vulnerability scanning section', async ({ page }) => {
    // Look for vulnerability or security section
    const vulnSection = page
      .locator('text=/vulnerabilit|security|cve|risk/i')
      .or(page.locator('[data-testid="vulnerability-card"]'))
      .first();

    await page.waitForTimeout(2000);

    // Vulnerability section should exist in dashboard or settings
    const hasSection = await vulnSection.isVisible().catch(() => false);

    if (!hasSection) {
      // Check in settings drawer
      const settingsButton = page.getByRole('button', { name: /settings/i }).first();
      await settingsButton.click();

      const vulnSettings = page.getByText(/vulnerabilit|security|scan/i).first();
      await expect(vulnSettings).toBeVisible({ timeout: 5000 });
    }
  });

  test('should show scan status', async ({ page }) => {
    // Look for scan status indicators
    const scanStatus = page
      .locator('text=/scanning|idle|complete|in progress|last scan/i')
      .or(page.locator('[data-testid="scan-status"]'))
      .first();

    await page.waitForTimeout(2000);

    const hasStatus = await scanStatus.isVisible().catch(() => false);
    const hasDisabled = await page
      .getByText(/disabled|not available/i)
      .isVisible()
      .catch(() => false);

    // Should show either status or disabled message
    expect(hasStatus || hasDisabled).toBeTruthy();
  });

  test('should display vulnerability count if available', async ({ page }) => {
    // Look for vulnerability count/summary
    const vulnCount = page
      .locator('text=/\\d+\\s*(vulnerabilit|issue|cve|finding)/i')
      .or(page.locator('[data-testid="vuln-count"]'))
      .first();

    await page.waitForTimeout(3000);

    const hasCount = await vulnCount.isVisible().catch(() => false);
    const hasNone = await page
      .getByText(/no vulnerabilit|no issue|clean|secure/i)
      .isVisible()
      .catch(() => false);

    // Either show count or "no vulnerabilities" message
    expect(hasCount || hasNone).toBeTruthy();
  });

  test('should have scan trigger button', async ({ page }) => {
    const scanButton = page
      .getByRole('button', { name: /scan|check|analyze/i })
      .or(page.locator('[data-testid="vuln-scan-btn"]'))
      .first();

    await page.waitForTimeout(2000);

    // Scan button may be in card or settings
    const hasButton = await scanButton.isVisible().catch(() => false);

    if (!hasButton) {
      // Open settings and check there
      const settingsButton = page.getByRole('button', { name: /settings/i }).first();
      await settingsButton.click();

      const settingsScanBtn = page.getByRole('button', { name: /scan|enable/i }).first();
      const hasScanInSettings = await settingsScanBtn.isVisible().catch(() => false);
      expect(hasScanInSettings).toBeTruthy();
    }
  });
});
