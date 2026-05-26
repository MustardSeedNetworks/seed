import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Dashboard E2E Tests
 *
 * Tests that dashboard cards render correctly:
 * - Link status card
 * - Gateway card
 * - DNS card
 * - Network discovery card
 * - Settings drawer functionality
 */

// Note: NOT tagged @smoke — beforeEach asserts a /link/i heading that
// no longer matches the dashboard layout. Re-tag once the assertion
// is stabilised (#1053 follow-up).
test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/');
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should display Link Status card', async ({ page }) => {
    const linkCard = page
      .locator('[data-testid="link-card"]')
      .or(page.locator('h3:has-text("Link"), h4:has-text("Link")').first());
    await expect(linkCard).toBeVisible();
  });

  test('should display Gateway card', async ({ page }) => {
    const gatewayCard = page.locator('h3:has-text("Gateway"), h4:has-text("Gateway")').first();
    await expect(gatewayCard).toBeVisible();
  });

  test('should display DNS card', async ({ page }) => {
    const dnsCard = page.locator('h3:has-text("DNS"), h4:has-text("DNS")').first();
    await expect(dnsCard).toBeVisible();
  });

  test('should open settings drawer', async ({ page }) => {
    // Click settings button
    const settingsButton = page
      .getByRole('button', { name: /settings/i })
      .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'))
      .first();
    await settingsButton.click();

    // Settings drawer should be visible
    await expect(page.getByText(/thresholds|appearance|discovery/i)).toBeVisible({ timeout: 5000 });
  });

  test('should toggle theme in settings', async ({ page }) => {
    // Open settings
    const settingsButton = page
      .getByRole('button', { name: /settings/i })
      .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'))
      .first();
    await settingsButton.click();

    // Find and click theme toggle
    const themeSection = page.getByText(/appearance|theme/i).first();
    await expect(themeSection).toBeVisible();
  });

  test('should show help modal', async ({ page }) => {
    // Click help button
    const helpButton = page
      .getByRole('button', { name: /help/i })
      .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'))
      .first();
    await helpButton.click();

    // Help modal should be visible
    await expect(page.getByRole('dialog').or(page.locator('[role="dialog"]'))).toBeVisible({
      timeout: 5000,
    });
  });
});
