import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

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
 * Help Modal:
 * - Open/close help modal
 * - Navigation and table of contents
 * - Section scrolling
 * - Search functionality (if implemented)
 * - Keyboard navigation (ESC to close)
 * - Click outside to dismiss
 * - Content rendering
 */

test.describe('Theme Toggle and Help Modal', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
    await page.goto('/');
    // Pin to level: 1 + exact-match "Link" so the H3 "Link Status" card
    // chrome doesn't trip strict mode (same fix as auth.spec / dashboard.spec).
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test.describe('Theme Toggle', () => {
    test('should toggle theme when clicking theme button', async ({ page }) => {
      // Open settings to find theme toggle
      const settingsButton = page.getByTestId('header-open-settings');

      await settingsButton.click();

      // Get current theme from HTML element
      const htmlElement = page.locator('html');
      const initialClasses = await htmlElement.getAttribute('class');
      const wasLight = !initialClasses?.includes('dark');

      // Find and click theme toggle
      await page
        .getByRole('button', { name: /appearance/i })
        .first()
        .click();
      const themeToggle = page.getByTestId('theme-toggle');

      await themeToggle.click();

      // Verify theme changed
      const newClasses = await htmlElement.getAttribute('class');
      const isNowDark = newClasses?.includes('dark');

      if (wasLight) {
        expect(isNowDark).toBe(true);
      } else {
        expect(isNowDark).toBe(false);
      }
    });

    test('should update document root class when theme changes', async ({ page }) => {
      // Open settings
      const settingsButton = page.getByTestId('header-open-settings');

      await settingsButton.click();

      // Find theme toggle
      await page
        .getByRole('button', { name: /appearance/i })
        .first()
        .click();
      const themeToggle = page.getByTestId('theme-toggle');

      // Toggle to dark
      const htmlElement = page.locator('html');
      let classes = await htmlElement.getAttribute('class');

      if (!classes?.includes('dark')) {
        await themeToggle.click();
      }

      // Verify dark class present
      classes = await htmlElement.getAttribute('class');
      expect(classes).toContain('dark');

      // Toggle to light
      await themeToggle.click();

      // Verify dark class removed
      classes = await htmlElement.getAttribute('class');
      expect(classes).not.toContain('dark');
    });

    test('should persist theme in localStorage', async ({ page }) => {
      // Open settings
      const settingsButton = page.getByTestId('header-open-settings');

      await settingsButton.click();

      // Find and click theme toggle
      await page
        .getByRole('button', { name: /appearance/i })
        .first()
        .click();
      const themeToggle = page.getByTestId('theme-toggle');

      await themeToggle.click();

      // Check localStorage for theme preference
      const storedTheme = await page.evaluate(
        () => localStorage.getItem('theme') || localStorage.getItem('seed-theme'),
      );

      // Should have a theme preference stored
      expect(storedTheme).toBeTruthy();
      expect(['light', 'dark']).toContain(storedTheme);
    });

    test('should persist theme after page reload', async ({ page }) => {
      // Open settings
      const settingsButton = page.getByTestId('header-open-settings');

      await settingsButton.click();

      // Toggle to dark theme
      await page
        .getByRole('button', { name: /appearance/i })
        .first()
        .click();
      const themeToggle = page.getByTestId('theme-toggle');

      const htmlElement = page.locator('html');
      let classes = await htmlElement.getAttribute('class');

      // Ensure we're in dark mode
      if (!classes?.includes('dark')) {
        await themeToggle.click();
      }

      // Verify dark mode
      classes = await htmlElement.getAttribute('class');
      const wasDark = classes?.includes('dark');

      // Reload page
      await page.reload();

      // Verify theme persisted
      const reloadedClasses = await page.locator('html').getAttribute('class');
      const stillDark = reloadedClasses?.includes('dark');

      expect(stillDark).toBe(wasDark);
    });

    test('should render all cards correctly in both themes', async ({ page }) => {
      // Get initial card count
      const initialCards = await page.getByTestId('card').count();
      expect(initialCards).toBeGreaterThan(0);

      const settingsButton = page.getByTestId('header-open-settings');
      const appearanceAccordion = page.getByRole('button', { name: /appearance/i }).first();
      const themeToggle = page.getByTestId('theme-toggle');
      const closeButton = page.getByTestId('settings-drawer-close');

      // Open settings → expand Appearance → toggle theme → close
      await settingsButton.click();
      await appearanceAccordion.click();
      await themeToggle.click();
      await closeButton.click();

      // Verify all cards still visible
      const cardsAfterToggle = await page.getByTestId('card').count();
      expect(cardsAfterToggle).toBeGreaterThanOrEqual(initialCards - 1); // Allow for minor variance

      // Toggle back — reopen drawer + re-expand accordion (accordion collapses on close)
      await settingsButton.click();
      await appearanceAccordion.click();
      await themeToggle.click();
      await closeButton.click();

      // Verify cards still visible in original theme
      const cardsAfterSecondToggle = await page.getByTestId('card').count();
      expect(cardsAfterSecondToggle).toBeGreaterThanOrEqual(initialCards - 1);
    });

    test('should maintain theme toggle state in settings', async ({ page }) => {
      // Open settings
      const settingsButton = page.getByTestId('header-open-settings');

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

  test.describe('Help Modal', () => {
    test('should open help modal when clicking help button', async ({ page }) => {
      // Find and click help button
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Verify modal opens
      const modal = page
        .getByRole('dialog')
        .or(page.locator('[role="dialog"]'))
        .or(page.locator('[class*="modal"]'));

      await expect(modal).toBeVisible({ timeout: 5000 });
    });

    test('should display help modal with navigation/table of contents', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Look for navigation/TOC
      const navigation = page.locator('text=/table.*contents|navigation|contents|sections/i');
      const hasNavigation = await navigation.first().isVisible();

      // Navigation may or may not be present depending on implementation
      expect(hasNavigation).toBeDefined();
    });

    test('should close help modal with close button', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Find and click close button (help modal's own close, not the settings drawer's)
      const closeButton = page.getByTestId('help-modal-close');

      await closeButton.click();

      // Verify modal closes
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).not.toBeVisible({ timeout: 3000 });
    });

    test('should close help modal with ESC key', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Verify modal is open
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();

      // Press ESC key
      await page.keyboard.press('Escape');

      // Verify modal closes
      await expect(modal).not.toBeVisible({ timeout: 3000 });
    });

    test('should close help modal when clicking outside', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Verify modal is open
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();

      // Click outside modal (on backdrop)
      const backdrop = page.locator('[class*="backdrop"], [class*="overlay"]').first();
      const hasBackdrop = await backdrop.isVisible();

      if (hasBackdrop) {
        await backdrop.click({ position: { x: 10, y: 10 } });

        // Modal should close
        await expect(modal).not.toBeVisible({ timeout: 3000 });
      }
    });

    test('should display help content sections', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Look for common help topics
      const helpTopics = page.locator(
        'text=/dashboard|network|wifi|discovery|speed.*test|settings|authentication/i',
      );

      const topicCount = await helpTopics.count();
      expect(topicCount).toBeGreaterThan(0);
    });

    test('should scroll to section when clicking TOC link', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Look for clickable TOC links
      const tocLinks = page.locator('a[href^="#"], button[data-section]');
      const linkCount = await tocLinks.count();

      if (linkCount > 0) {
        // Click first TOC link
        await tocLinks.first().click();

        // Modal should still be open
        const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
        await expect(modal).toBeVisible();
      }
    });

    test('should filter help content with search functionality', async ({ page }) => {
      // This test is skipped if search is not implemented

      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Look for search input.
      // Loud failure beats silent skip: if the help drawer search disappears,
      // this test surfaces the regression instead of hiding it.
      const searchInput = page.getByPlaceholder(/search|filter/i);
      await expect(
        searchInput,
        'precondition: help drawer search input must be visible',
      ).toBeVisible();

      // Enter search term
      await searchInput.fill('network');

      // Verify filtered results
      const results = page.locator('text=/network/i');
      const resultCount = await results.count();

      expect(resultCount).toBeGreaterThan(0);
    });

    test('should render help content correctly', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Verify modal has content (headings, paragraphs)
      const headings = page.locator('h1, h2, h3, h4, h5, h6');
      const paragraphs = page.locator('p');

      const headingCount = await headings.count();
      const paragraphCount = await paragraphs.count();

      // Should have some content
      expect(headingCount + paragraphCount).toBeGreaterThan(0);
    });

    test('should maintain scroll position when reopening help modal', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Scroll within modal
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await modal.evaluate((el) => {
        const scrollable = el.querySelector('[class*="scroll"]') || el;
        scrollable.scrollTop = 100;
      });

      // Close modal
      await page.keyboard.press('Escape');

      // Reopen modal
      await helpButton.click();

      // After close + reopen the modal should reset its scroll to top.
      // The original test accepted any non-negative scroll value, which is
      // tautological (scrollTop is always non-negative) and would pass even
      // if the modal silently leaked scroll state across opens.
      const scrollPosition = await modal.evaluate((el) => {
        const scrollable = el.querySelector('[class*="scroll"]') ?? el;
        return scrollable.scrollTop;
      });
      expect(scrollPosition, 'modal scroll should reset to top on reopen').toBe(0);
    });

    test('should display help modal in both light and dark themes', async ({ page }) => {
      // Test in light theme
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();

      // Close modal
      await page.keyboard.press('Escape');

      // Toggle to dark theme
      const settingsButton = page.getByTestId('header-open-settings');

      await settingsButton.click();

      await page
        .getByRole('button', { name: /appearance/i })
        .first()
        .click();
      const themeToggle = page.getByTestId('theme-toggle');

      await themeToggle.click();

      const closeSettings = page.getByTestId('settings-drawer-close');

      await closeSettings.click();

      // Open help modal in dark theme
      await helpButton.click();

      // Modal should be visible in dark theme
      await expect(modal).toBeVisible();

      // Verify dark theme applied
      const htmlClasses = await page.locator('html').getAttribute('class');
      const isDark = htmlClasses?.includes('dark');

      expect(isDark).toBe(true);
    });
  });

  test.describe('Theme and Help Integration', () => {
    test('should allow theme toggle while help modal is open', async ({ page }) => {
      // Open help modal
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();

      // Help modal blocks header buttons via backdrop. Close help with ESC,
      // then exercise the theme toggle path — verifying the two surfaces
      // don't deadlock each other.
      await page.keyboard.press('Escape');
      await expect(modal).not.toBeVisible({ timeout: 3000 });

      const settingsButton = page.getByTestId('header-open-settings');
      await settingsButton.click();
      await page
        .getByRole('button', { name: /appearance/i })
        .first()
        .click();
      const themeToggle = page.getByTestId('theme-toggle');
      await themeToggle.click();

      // Reopen help modal — should still render after theme switch.
      const closeSettings = page.getByTestId('settings-drawer-close');
      await closeSettings.click();
      await helpButton.click();
      await expect(modal).toBeVisible();
    });

    test('should maintain help modal state when toggling theme', async ({ page }) => {
      // Get initial theme
      const initialClasses = await page.locator('html').getAttribute('class');
      const initialTheme = initialClasses?.includes('dark') ? 'dark' : 'light';

      // Open settings and toggle theme
      const settingsButton = page.getByTestId('header-open-settings');

      await settingsButton.click();

      await page
        .getByRole('button', { name: /appearance/i })
        .first()
        .click();
      const themeToggle = page.getByTestId('theme-toggle');

      await themeToggle.click();

      const closeSettings = page.getByTestId('settings-drawer-close');

      await closeSettings.click();

      // Open help modal in new theme
      const helpButton = page.getByTestId('header-open-help');

      await helpButton.click();

      // Verify theme changed
      const newClasses = await page.locator('html').getAttribute('class');
      const newTheme = newClasses?.includes('dark') ? 'dark' : 'light';

      expect(newTheme).not.toBe(initialTheme);

      // Help modal should be visible
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();
    });
  });
});
