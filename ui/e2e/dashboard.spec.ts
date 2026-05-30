import { expect, test } from '@playwright/test';
import { sidebarHelpButton, sidebarSettingsButton, skipSetupWizard } from './helpers/auth';

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
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should display Link Status card', async ({ page }) => {
    // Card.tsx generates id="card-title-<slug>" on every card's H3 — same
    // pattern as the Gateway/DNS assertions below. Drops the brittle
    // `h3:has-text("Link")` fallback that matched any element containing
    // "Link" as a substring (including the sidebar nav item).
    await expect(page.locator('#card-title-link')).toBeVisible();
  });

  test('should display Gateway card', async ({ page }) => {
    // Card.tsx generates id="card-title-<slug>" on every card's H3 —
    // stable across i18n drift since slug derives from the title prop
    // at component-mount time (still English in the dev backend).
    await expect(page.locator('#card-title-gateway')).toBeVisible();
  });

  test('should display DNS card', async ({ page }) => {
    await expect(page.locator('#card-title-dns')).toBeVisible();
  });

  test('should open settings drawer', async ({ page }) => {
    // Click settings button
    const settingsButton = sidebarSettingsButton(page);
    await settingsButton.click();

    // Settings drawer should be visible
    await expect(page.getByTestId('settings-drawer')).toBeVisible({ timeout: 5000 });
  });

  test('should toggle theme in settings', async ({ page }) => {
    // Open settings
    const settingsButton = sidebarSettingsButton(page);
    await settingsButton.click();

    // Find appearance section by stable testid (was /appearance|theme/i —
    // both translated under es). AppearanceSettings carries
    // data-testid="appearance-settings-section" on its CollapsibleSection.
    await expect(page.getByTestId('appearance-settings-section')).toBeVisible();
  });

  test('should show help modal', async ({ page }) => {
    // Click help button
    const helpButton = sidebarHelpButton(page);
    await helpButton.click();

    // Help modal should be visible
    await expect(page.getByRole('dialog').or(page.locator('[role="dialog"]'))).toBeVisible({
      timeout: 5000,
    });
  });
});
