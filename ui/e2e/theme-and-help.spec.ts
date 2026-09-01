import { expect, type Page, test } from '@playwright/test';
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
 * - Stored preference applied on load and left unmodified
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

/** Theme modes accepted by the `seed-theme` key (hooks/useTheme.ts). */
type Theme = 'light' | 'dark' | 'system';

/**
 * Pre-seed the `seed-theme` preference so the very first navigation already
 * carries it. Must be called before page.goto.
 *
 * This replaces an `evaluate(setItem)` + `page.reload()` pattern that was
 * intermittently failing on WebKit with "WebKit encountered an internal error"
 * out of `page.reload`, ejecting PRs from the merge queue. The localStorage
 * write was never the race — `page.evaluate` awaits a synchronous `setItem`,
 * so the value was committed before the reload was issued. Two things were
 * wrong underneath:
 *
 *  1. `disableAnimations()` threw at document start (see helpers/auth.ts), so
 *     the suite's animations were never actually disabled. Every reload here
 *     landed on a page mid-transition — the app-wide re-theme plus the drawer
 *     transitions — which is what WebKit turned into an internal error.
 *  2. Each reload fired the moment the header painted, while the dashboard's
 *     data fetches, the /ws socket and the `skipSetupWizard` route
 *     interception were all still in flight.
 *
 * The helper fix addresses (1). Seeding before the first navigation addresses
 * (2) by removing the second navigation altogether: every theme test below is
 * one page, one `goto`, with nothing left in flight to abort — which is also
 * how a real operator meets the app, with the preference already in storage
 * when the document loads.
 *
 * Note this must be called *after* `disableAnimations()` has been fixed to
 * stop throwing: on WebKit an init script that throws aborts the init scripts
 * registered after it, which silently dropped this write.
 */
async function seedTheme(page: Page, theme: Theme): Promise<void> {
  await page.addInitScript((value) => {
    localStorage.setItem('seed-theme', value);
  }, theme);
}

/** Load the dashboard and wait for it to mount. */
async function gotoDashboard(page: Page): Promise<void> {
  await page.goto('/');
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
}

/** Assert the document root reflects the given effective theme. */
async function expectTheme(page: Page, theme: 'light' | 'dark'): Promise<void> {
  const html = page.locator('html');
  if (theme === 'dark') {
    await expect(html).toHaveClass(/dark/);
  } else {
    await expect(html).not.toHaveClass(/dark/);
  }
}

test.describe('Theme Toggle and Help Modal', { tag: '@smoke' }, () => {
  // Run this file's tests sequentially in a single worker. Toggling the theme
  // re-renders every theme consumer app-wide; under the 2-worker CI split the
  // resulting CPU contention stalls the toggle/close clicks past the 30s
  // timeout. These tests pass reliably one-at-a-time.
  test.describe.configure({ mode: 'serial' });

  // The fixture page is prepared here but deliberately left un-navigated:
  // theme tests must seed `seed-theme` before the first navigation, so each
  // one owns its single page.goto. Tests that do not care about the theme
  // navigate via the describe-level beforeEach below.
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await disableAnimations(page);
  });

  test.describe('Theme Toggle', () => {
    // Theme is driven by the `seed-theme` localStorage key (hooks/useTheme.ts).
    // We seed it directly rather than driving the settings drawer's theme
    // <select>: that control sits ~10 sections deep in a scrollable drawer
    // and, under the 2-worker CI split, Playwright's scroll-into-view stalls
    // past the test timeout. The drawer's <select> binding itself is covered
    // in settings.spec.ts (full E2E tier). These assert the behavior that
    // actually matters for smoke: the app reads, applies and keeps the stored
    // preference, and the dashboard renders under either theme.
    //
    // One test per theme rather than one test walking both, so that each needs
    // exactly one navigation — see seedTheme() for why that matters.
    for (const theme of ['dark', 'light'] as const) {
      test(`should apply and keep the saved ${theme} theme preference`, async ({ page }) => {
        await seedTheme(page, theme);
        await gotoDashboard(page);

        await expectTheme(page, theme);
        // The app applies the stored preference without rewriting it, so a
        // later load finds the same value and resolves to the same theme.
        expect(await page.evaluate(() => localStorage.getItem('seed-theme'))).toBe(theme);
      });

      test(`should keep cards rendered in the ${theme} theme`, async ({ page }) => {
        await seedTheme(page, theme);
        await gotoDashboard(page);

        await expectTheme(page, theme);
        // Cards mount as their data resolves, so poll rather than count once.
        await expect.poll(() => page.getByTestId('card').count()).toBeGreaterThan(0);
      });
    }

    test('should maintain theme toggle state in settings', async ({ page }) => {
      await gotoDashboard(page);

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
      // explicitly and uses Playwright's colorScheme emulation to
      // drive both branches deterministically. The emulation changes
      // need no navigation: the hook's matchMedia listener re-themes
      // the live document.
      await page.emulateMedia({ colorScheme: 'dark' });
      await seedTheme(page, 'system');
      await gotoDashboard(page);
      await expectTheme(page, 'dark');

      // Live system theme change: app should follow.
      await page.emulateMedia({ colorScheme: 'light' });
      await expectTheme(page, 'light');

      // Back to dark to confirm the listener is bidirectional.
      await page.emulateMedia({ colorScheme: 'dark' });
      await expectTheme(page, 'dark');
    });
  });

  test.describe('Help Drawer', () => {
    test.beforeEach(async ({ page }) => {
      await gotoDashboard(page);
    });

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
      await searchInput.fill('wifi troubleshooting');
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

      // The default (About) section renders real product prose (the help
      // glossary's botanical module terms were retired with the function-first
      // nav — this asserts stable About copy instead).
      await expect(content).toContainText('network diagnostics');
      await expect(content).toContainText('Mustard Seed Networks');

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
  });

  // Separate from the Help Drawer describe above because these seed a theme
  // before the first navigation, so they cannot inherit its navigating
  // beforeEach. One test per theme keeps each to a single page.goto.
  test.describe('Help Drawer theming', () => {
    for (const theme of ['light', 'dark'] as const) {
      test(`should display help drawer in the ${theme} theme`, async ({ page }) => {
        await seedTheme(page, theme);
        await gotoDashboard(page);
        await expectTheme(page, theme);

        const drawer = page.getByTestId('help-drawer');
        await sidebarHelpButton(page).click();
        await expect(drawer).toBeVisible();

        await page.keyboard.press('Escape');
        await expect(drawer).not.toBeVisible();
      });
    }
  });
});
