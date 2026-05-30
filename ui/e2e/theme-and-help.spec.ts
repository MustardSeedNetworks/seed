import { expect, test } from '@playwright/test';
import {
  disableAnimations,
  sidebarHelpButton,
  sidebarSettingsButton,
  skipSetupWizard,
} from './helpers/auth';

/**
 * Theme Toggle and Help Modal E2E Tests
 *
 * Comprehensive tests for theme management and help system:
 *
 * Theme Toggle:
 * - Toggle between light and dark themes
 * - Verify document root class changes
 * - Theme persistence in localStorage
 * - Theme persistence after page reload
 * - Cards render correctly in both themes
 * - System theme preference (if implemented)
 *
 * Help Drawer:
 * - Open/close help drawer
 * - Navigation and table of contents
 * - Section switching
 * - Search functionality
 * - Keyboard navigation (ESC to close)
 * - Click outside to dismiss
 * - Real content rendering (bug-fix regression)
 */

test.describe('Theme Toggle and Help Modal', { tag: '@smoke' }, () => {
  // Run this file's tests sequentially in a single worker. Toggling the theme
  // re-renders every theme consumer app-wide; under the 2-worker CI split the
  // resulting CPU contention stalls the toggle/close clicks past the 30s
  // timeout. These tests pass reliably one-at-a-time.
  test.describe.configure({ mode: 'serial' });

  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await disableAnimations(page);
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test.describe('Theme Toggle', () => {
    test('should apply and persist the saved theme preference', async ({ page }) => {
      // Theme is driven by the `seed-theme` localStorage key (hooks/useTheme.ts).
      // We set it directly and reload rather than driving the settings drawer's
      // theme <select>: that control sits ~10 sections deep in a scrollable
      // drawer and, under the 2-worker CI split, Playwright's scroll-into-view
      // stalls past the test timeout. The drawer's <select> binding itself is
      // covered in settings.spec.ts (full E2E tier). This asserts the behavior
      // that actually matters for smoke: the app reads, applies, and persists
      // the stored preference.
      const html = page.locator('html');

      await page.evaluate(() => localStorage.setItem('seed-theme', 'dark'));
      await page.reload();
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      await expect(html).toHaveClass(/dark/);

      await page.evaluate(() => localStorage.setItem('seed-theme', 'light'));
      await page.reload();
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      await expect(html).not.toHaveClass(/dark/);

      // Preference survives a further reload with no change.
      await page.reload();
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      await expect(html).not.toHaveClass(/dark/);
    });

    test('should keep cards rendered in both themes', async ({ page }) => {
      const html = page.locator('html');
      const initialCards = await page.getByTestId('card').count();
      expect(initialCards).toBeGreaterThan(0);

      // Drive the theme deterministically via localStorage + reload rather
      // than through the settings drawer: this test asserts the *dashboard*
      // renders in both themes, and the drawer's close button is momentarily
      // unresponsive during the app-wide re-theme (the toggle UI itself is
      // covered by the dedicated theme-toggle tests above).
      await page.evaluate(() => localStorage.setItem('seed-theme', 'dark'));
      await page.reload();
      await expect(html).toHaveClass(/dark/);
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      expect(await page.getByTestId('card').count()).toBeGreaterThanOrEqual(initialCards - 1);

      await page.evaluate(() => localStorage.setItem('seed-theme', 'light'));
      await page.reload();
      await expect(html).not.toHaveClass(/dark/);
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      expect(await page.getByTestId('card').count()).toBeGreaterThanOrEqual(initialCards - 1);
    });

    test('should maintain theme toggle state in settings', async ({ page }) => {
      // Open settings
      const settingsButton = sidebarSettingsButton(page);

      await settingsButton.click();

      // Get current theme
      const htmlClasses = await page.locator('html').getAttribute('class');
      const isDark = htmlClasses?.includes('dark') ?? false;

      // Close and reopen settings
      const closeButton = page.getByTestId('settings-drawer-close');

      await closeButton.click();

      await settingsButton.click();

      // Theme should still be the same
      const reopenedClasses = await page.locator('html').getAttribute('class');
      const stillDark = reopenedClasses?.includes('dark') ?? false;

      expect(stillDark).toBe(isDark);
    });

    test('should track system theme when theme=system', async ({ page }) => {
      // System theme detection IS implemented in seed (see
      // ui/src/hooks/useTheme.ts — Theme = 'light' | 'dark' |
      // 'system', live matchMedia listener). Exercises both
      // system → dark and system → light branches and the
      // bidirectional matchMedia 'change' listener.
      //
      // Previous shape loaded the page with the default theme
      // (hardcoded 'dark') and compared the resulting html class
      // against window.matchMedia — only passed when the host
      // happened to also be dark. This rewrite forces theme=system
      // explicitly via localStorage and uses Playwright's
      // colorScheme emulation to drive both branches
      // deterministically.

      // Emulate dark system preference and load with theme=system.
      await page.emulateMedia({ colorScheme: 'dark' });
      await page.evaluate(() => localStorage.setItem('seed-theme', 'system'));
      await page.reload();
      await expect(page.locator('html')).toHaveClass(/dark/);

      // Live system theme change: app should follow.
      await page.emulateMedia({ colorScheme: 'light' });
      await expect(page.locator('html')).not.toHaveClass(/dark/);

      // Back to dark to confirm the listener is bidirectional.
      await page.emulateMedia({ colorScheme: 'dark' });
      await expect(page.locator('html')).toHaveClass(/dark/);
    });
  });

  test.describe('Help Drawer', () => {
    test('should open help drawer when clicking help button', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      // The data-driven drawer is the canonical target (data-testid="help-drawer").
      const drawer = page.getByTestId('help-drawer');
      await expect(drawer).toBeVisible({ timeout: 5000 });
      await expect(drawer).toHaveAttribute('role', 'dialog');
      await expect(drawer).toHaveAttribute('aria-modal', 'true');
    });

    test('should display help drawer with navigation/table of contents', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const drawer = page.getByTestId('help-drawer');
      await expect(drawer).toBeVisible();

      // The drawer lists navigable sections in its sidebar nav.
      const tocButtons = drawer.locator('nav button');
      expect(
        await tocButtons.count(),
        'help drawer should list navigable sections',
      ).toBeGreaterThan(0);
    });

    test('should close help drawer with close button', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const drawer = page.getByTestId('help-drawer');
      await expect(drawer).toBeVisible();

      // The drawer's own close button (not the settings drawer's).
      await page.getByTestId('help-drawer-close').click();

      await expect(drawer).not.toBeVisible({ timeout: 3000 });
    });

    test('should close help drawer with ESC key', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const drawer = page.getByTestId('help-drawer');
      await expect(drawer).toBeVisible();

      await page.keyboard.press('Escape');

      await expect(drawer).not.toBeVisible({ timeout: 3000 });
    });

    test('should close help drawer when clicking outside', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const drawer = page.getByTestId('help-drawer');
      await expect(drawer).toBeVisible();

      // Click the backdrop (dark overlay behind the drawer). The
      // previous `[class*="backdrop"]` substring-match against
      // Tailwind utilities was unreliable AND was wrapped in an
      // `if (await backdrop.isVisible())` gate that silently passed
      // when the locator missed — flagged by the cleanup audit as
      // the last hidden-failure test in seed E2E.
      const backdrop = page.getByTestId('help-drawer-backdrop');
      await expect(backdrop).toBeVisible();
      await backdrop.click({ position: { x: 10, y: 10 } });
      await expect(drawer).not.toBeVisible({ timeout: 3000 });
    });

    test('should switch sections when clicking a table-of-contents entry', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const drawer = page.getByTestId('help-drawer');
      await expect(drawer).toBeVisible();

      const tocButtons = drawer.locator('nav button');
      expect(
        await tocButtons.count(),
        'help drawer should list navigable sections',
      ).toBeGreaterThan(1);

      // Selecting a section keeps the drawer open and swaps the content pane.
      const content = page.getByTestId('help-drawer-content');
      const before = await content.innerText();
      await tocButtons.nth(1).click();
      await expect(drawer).toBeVisible();
      // The active section's title heads the content pane and changes on switch.
      await expect.poll(async () => content.innerText()).not.toBe(before);
    });

    test('should filter help content with search functionality', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const drawer = page.getByTestId('help-drawer');
      await expect(drawer).toBeVisible();

      // Loud failure beats silent skip: if the help drawer search disappears,
      // this test surfaces the regression instead of hiding it.
      const searchInput = drawer.getByPlaceholder(/search|filter/i);
      await expect(
        searchInput,
        'precondition: help drawer search input must be visible',
      ).toBeVisible();

      // Narrow to a single section, then confirm the TOC shrank.
      const tocButtons = drawer.locator('nav button');
      const allCount = await tocButtons.count();
      await searchInput.fill('wifi survey');
      await expect.poll(async () => tocButtons.count()).toBeLessThan(allCount);
      expect(await tocButtons.count()).toBeGreaterThan(0);
    });

    test('should render real help content (bug-fix regression)', async ({ page }) => {
      // The old modal defined every section with `content: null`, so the pane
      // rendered nothing. This asserts the drawer renders actual body copy —
      // the core bug this feature fixes.
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const content = page.getByTestId('help-drawer-content');
      await expect(content).toBeVisible();

      // The default (About) section names the Seed modules in its body.
      await expect(content).toContainText('Roots');
      await expect(content).toContainText('Canopy');

      // And there is substantive prose, not an empty pane.
      const text = (await content.innerText()).trim();
      expect(text.length, 'content pane should not be empty').toBeGreaterThan(100);
      expect(await content.locator('p').count()).toBeGreaterThan(0);
    });

    test('should reset content scroll to top on reopen', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      await helpButton.click();

      const content = page.getByTestId('help-drawer-content');
      await expect(content).toBeVisible();
      await content.evaluate((el) => {
        el.scrollTop = 100;
      });

      await page.keyboard.press('Escape');
      await expect(page.getByTestId('help-drawer')).not.toBeVisible();

      await helpButton.click();
      await expect(content).toBeVisible();

      // The content pane remounts on reopen, so scroll resets to the top.
      const scrollPosition = await content.evaluate((el) => el.scrollTop);
      expect(scrollPosition, 'content scroll should reset to top on reopen').toBe(0);
    });

    test('should display help drawer in light and dark themes', async ({ page }) => {
      const helpButton = sidebarHelpButton(page);
      const drawer = page.getByTestId('help-drawer');
      const html = page.locator('html');

      await page.evaluate(() => localStorage.setItem('seed-theme', 'light'));
      await page.reload();
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      await expect(html).not.toHaveClass(/dark/);
      await helpButton.click();
      await expect(drawer).toBeVisible();
      await page.keyboard.press('Escape');
      await expect(drawer).not.toBeVisible();

      await page.evaluate(() => localStorage.setItem('seed-theme', 'dark'));
      await page.reload();
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      await expect(html).toHaveClass(/dark/);
      await helpButton.click();
      await expect(drawer).toBeVisible();
    });
  });
});
