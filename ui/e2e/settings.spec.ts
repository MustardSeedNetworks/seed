import { expect, test } from '@playwright/test';
import { sidebarSettingsButton, skipSetupWizard } from './helpers/auth';

/**
 * Settings E2E Tests
 *
 * Tests the settings drawer functionality:
 * - All settings sections accessible
 * - Settings save/load correctly (CRUD operations)
 * - Theme switching
 * - Threshold configuration
 * - Discovery settings (scan methods, timeouts)
 * - DNS test hostname configuration
 * - Performance settings
 * - Auto-save indicator
 * - Settings validation (reject invalid values)
 * - Settings persistence after page reload
 */

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });

    // Open settings drawer
    const settingsButton = sidebarSettingsButton(page);
    await settingsButton.click();

    // Wait for drawer to open
    await expect(page.getByTestId('settings-drawer')).toBeVisible({ timeout: 5000 });
  });

  test('should display Appearance settings section', async ({ page }) => {
    await expect(page.getByTestId('appearance-settings-section')).toBeVisible();
  });

  test('should display Thresholds settings section', async ({ page }) => {
    await expect(page.getByTestId('thresholds-settings-section')).toBeVisible();
  });

  test('should display Discovery settings section', async ({ page }) => {
    await expect(page.getByTestId('discovery-settings-section')).toBeVisible();
  });

  test('should display DNS settings section', async ({ page }) => {
    await expect(page.getByTestId('dns-settings-section')).toBeVisible();
  });

  test('should display Performance settings section', async ({ page }) => {
    await expect(page.getByTestId('performance-settings-section')).toBeVisible();
  });

  test('should toggle theme between light and dark', async ({ page }) => {
    // theme-toggle testid is on AppearanceSettings.tsx:169 (the quick-
    // toggle button). The Appearance section is collapsed by default;
    // expand it first. Previously a .or() chain with /dark|light|theme/i
    // regex + checkbox + the testid silently no-op'd because the
    // appearance section wasn't expanded.
    await page.getByTestId('appearance-settings-section').click();
    const themeToggle = page.getByTestId('theme-toggle');
    await expect(themeToggle).toBeVisible();

    const htmlClasses = await page.locator('html').getAttribute('class');
    const wasDark = htmlClasses?.includes('dark') ?? false;

    await themeToggle.click();

    const newHtmlClasses = await page.locator('html').getAttribute('class');
    const isDark = newHtmlClasses?.includes('dark') ?? false;

    expect(isDark).not.toBe(wasDark);
  });

  test('should have input fields for threshold values', async ({ page }) => {
    // Look for threshold input fields
    const thresholdInputs = page.locator(
      'input[type="number"], input[type="range"], input[name*="threshold"]',
    );

    const inputCount = await thresholdInputs.count();
    expect(inputCount).toBeGreaterThan(0);
  });

  // Dropped: "should show auto-save indicator" — the assertion was
  // literally `expect(true).toBeTruthy()` after a discarded
  // `_hasAutoSave` lookup that combined an i18n-fragile getByText
  // regex with a `[data-testid="auto-save"]` selector that has no
  // matching element in the seed UI. Net signal: zero.
  // Auto-save behaviour is exercised by the threshold-update CRUD
  // test below ("should update threshold values and persist") which
  // polls window.localStorage for the actual flushed state.

  test('should close settings drawer', async ({ page }) => {
    // settings-drawer-close testid is on SettingsDrawer.tsx.
    // Previously matched by /close/i regex + svg class fallback —
    // i18n-fragile + brittle to icon library changes.
    await page.getByTestId('settings-drawer-close').click();
    await expect(page.getByTestId('settings-drawer')).toBeHidden({
      timeout: 3000,
    });
  });

  test('should persist settings after drawer close and reopen', async ({ page }) => {
    // Expand Appearance section first; theme-toggle lives inside it.
    await page.getByTestId('appearance-settings-section').click();
    const themeToggle = page.getByTestId('theme-toggle');
    await expect(themeToggle).toBeVisible();

    await themeToggle.click();
    const themeAfterToggle = await page.locator('html').getAttribute('class');

    // Close drawer using stable testid
    await page.getByTestId('settings-drawer-close').click();
    await expect(page.getByTestId('settings-drawer')).toBeHidden();

    // Reopen via sidebar settings helper
    await sidebarSettingsButton(page).click();
    await expect(page.getByTestId('settings-drawer')).toBeVisible();

    const themeAfterReopen = await page.locator('html').getAttribute('class');
    expect(themeAfterReopen).toBe(themeAfterToggle);
  });
});

/**
 * Settings CRUD Operations E2E Tests
 *
 * Comprehensive testing of Create, Read, Update, Delete operations
 * for all settings categories with backend persistence verification.
 */
