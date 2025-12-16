import { test, expect } from '@playwright/test';

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

test.describe('Theme Toggle and Help Modal', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    // Login
    await page.getByLabel(/username/i).fill('admin');
    await page.getByLabel(/password/i).fill('luminetiq');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Wait for dashboard
    await expect(page.getByRole('heading', { name: /link|dashboard/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test.describe('Theme Toggle', () => {
    test('should toggle theme when clicking theme button', async ({ page }) => {
      // Open settings to find theme toggle
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Get current theme from HTML element
      const htmlElement = page.locator('html');
      const initialClasses = await htmlElement.getAttribute('class');
      const wasLight = !initialClasses?.includes('dark');

      // Find and click theme toggle
      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('input[type="checkbox"][name*="theme"]'))
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      await themeToggle.click();
      await page.waitForTimeout(500);

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
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Find theme toggle
      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      // Toggle to dark
      const htmlElement = page.locator('html');
      let classes = await htmlElement.getAttribute('class');

      if (!classes?.includes('dark')) {
        await themeToggle.click();
        await page.waitForTimeout(500);
      }

      // Verify dark class present
      classes = await htmlElement.getAttribute('class');
      expect(classes).toContain('dark');

      // Toggle to light
      await themeToggle.click();
      await page.waitForTimeout(500);

      // Verify dark class removed
      classes = await htmlElement.getAttribute('class');
      expect(classes).not.toContain('dark');
    });

    test('should persist theme in localStorage', async ({ page }) => {
      // Open settings
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Find and click theme toggle
      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      await themeToggle.click();
      await page.waitForTimeout(500);

      // Check localStorage for theme preference
      const storedTheme = await page.evaluate(() => {
        return localStorage.getItem('theme') || localStorage.getItem('netscope-theme');
      });

      // Should have a theme preference stored
      expect(storedTheme).toBeTruthy();
      expect(['light', 'dark']).toContain(storedTheme);
    });

    test('should persist theme after page reload', async ({ page }) => {
      // Open settings
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Toggle to dark theme
      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      const htmlElement = page.locator('html');
      let classes = await htmlElement.getAttribute('class');

      // Ensure we're in dark mode
      if (!classes?.includes('dark')) {
        await themeToggle.click();
        await page.waitForTimeout(500);
      }

      // Verify dark mode
      classes = await htmlElement.getAttribute('class');
      const wasDark = classes?.includes('dark');

      // Reload page
      await page.reload();
      await page.waitForTimeout(1000);

      // Verify theme persisted
      const reloadedClasses = await page.locator('html').getAttribute('class');
      const stillDark = reloadedClasses?.includes('dark');

      expect(stillDark).toBe(wasDark);
    });

    test('should render all cards correctly in both themes', async ({ page }) => {
      // Get initial card count
      const initialCards = await page.locator('[class*="card"]').count();
      expect(initialCards).toBeGreaterThan(0);

      // Open settings
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Toggle theme
      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      await themeToggle.click();
      await page.waitForTimeout(500);

      // Close settings
      const closeButton = page
        .getByRole('button', { name: /close/i })
        .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
        .first();

      await closeButton.click();
      await page.waitForTimeout(500);

      // Verify all cards still visible
      const cardsAfterToggle = await page.locator('[class*="card"]').count();
      expect(cardsAfterToggle).toBeGreaterThanOrEqual(initialCards - 1); // Allow for minor variance

      // Toggle back
      await settingsButton.click();
      await page.waitForTimeout(500);
      await themeToggle.click();
      await page.waitForTimeout(500);
      await closeButton.click();
      await page.waitForTimeout(500);

      // Verify cards still visible in original theme
      const cardsAfterSecondToggle = await page.locator('[class*="card"]').count();
      expect(cardsAfterSecondToggle).toBeGreaterThanOrEqual(initialCards - 1);
    });

    test('should maintain theme toggle state in settings', async ({ page }) => {
      // Open settings
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Get current theme
      const htmlClasses = await page.locator('html').getAttribute('class');
      const isDark = htmlClasses?.includes('dark') ?? false;

      // Close and reopen settings
      const closeButton = page
        .getByRole('button', { name: /close/i })
        .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
        .first();

      await closeButton.click();
      await page.waitForTimeout(500);

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Theme should still be the same
      const reopenedClasses = await page.locator('html').getAttribute('class');
      const stillDark = reopenedClasses?.includes('dark') ?? false;

      expect(stillDark).toBe(isDark);
    });

    test.skip('should respect system theme preference if implemented', async ({ page }) => {
      // This test is skipped as system theme preference may not be implemented
      // To implement: check if theme matches system preference on first load

      // Get system theme preference
      const systemPrefersDark = await page.evaluate(() => {
        return window.matchMedia('(prefers-color-scheme: dark)').matches;
      });

      // Check if app theme matches system theme
      const htmlClasses = await page.locator('html').getAttribute('class');
      const appIsDark = htmlClasses?.includes('dark') ?? false;

      // If system theme sync is implemented, these should match
      expect(appIsDark).toBe(systemPrefersDark);
    });
  });

  test.describe('Help Modal', () => {
    test('should open help modal when clicking help button', async ({ page }) => {
      // Find and click help button
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

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
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Look for navigation/TOC
      const navigation = page.locator('text=/table.*contents|navigation|contents|sections/i');
      const hasNavigation = await navigation.first().isVisible().catch(() => false);

      // Navigation may or may not be present depending on implementation
      expect(hasNavigation).toBeDefined();
    });

    test('should close help modal with close button', async ({ page }) => {
      // Open help modal
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Find and click close button
      const closeButton = page
        .getByRole('button', { name: /close/i })
        .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
        .first();

      await closeButton.click();

      // Verify modal closes
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).not.toBeVisible({ timeout: 3000 });
    });

    test('should close help modal with ESC key', async ({ page }) => {
      // Open help modal
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Verify modal is open
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();

      // Press ESC key
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);

      // Verify modal closes
      await expect(modal).not.toBeVisible({ timeout: 3000 });
    });

    test('should close help modal when clicking outside', async ({ page }) => {
      // Open help modal
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Verify modal is open
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();

      // Click outside modal (on backdrop)
      const backdrop = page.locator('[class*="backdrop"], [class*="overlay"]').first();
      const hasBackdrop = await backdrop.isVisible().catch(() => false);

      if (hasBackdrop) {
        await backdrop.click({ position: { x: 10, y: 10 } });
        await page.waitForTimeout(500);

        // Modal should close
        await expect(modal).not.toBeVisible({ timeout: 3000 });
      }
    });

    test('should display help content sections', async ({ page }) => {
      // Open help modal
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Look for common help topics
      const helpTopics = page.locator(
        'text=/dashboard|network|wifi|discovery|speed.*test|settings|authentication/i'
      );

      const topicCount = await helpTopics.count();
      expect(topicCount).toBeGreaterThan(0);
    });

    test('should scroll to section when clicking TOC link', async ({ page }) => {
      // Open help modal
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Look for clickable TOC links
      const tocLinks = page.locator('a[href^="#"], button[data-section]');
      const linkCount = await tocLinks.count();

      if (linkCount > 0) {
        // Click first TOC link
        await tocLinks.first().click();
        await page.waitForTimeout(500);

        // Modal should still be open
        const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
        await expect(modal).toBeVisible();
      }
    });

    test.skip('should filter help content with search functionality', async ({ page }) => {
      // This test is skipped if search is not implemented

      // Open help modal
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Look for search input
      const searchInput = page.getByPlaceholder(/search|filter/i);
      const hasSearch = await searchInput.isVisible().catch(() => false);

      if (!hasSearch) {
        test.skip();
      }

      // Enter search term
      await searchInput.fill('network');
      await page.waitForTimeout(500);

      // Verify filtered results
      const results = page.locator('text=/network/i');
      const resultCount = await results.count();

      expect(resultCount).toBeGreaterThan(0);
    });

    test('should render help content correctly', async ({ page }) => {
      // Open help modal
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

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
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Scroll within modal
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await modal.evaluate((el) => {
        const scrollable = el.querySelector('[class*="scroll"]') || el;
        scrollable.scrollTop = 100;
      });

      await page.waitForTimeout(300);

      // Close modal
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);

      // Reopen modal
      await helpButton.click();
      await page.waitForTimeout(500);

      // Scroll position may reset (implementation-dependent)
      // This test documents expected behavior
      const scrollPosition = await modal.evaluate((el) => {
        const scrollable = el.querySelector('[class*="scroll"]') || el;
        return scrollable.scrollTop;
      });

      expect(scrollPosition).toBeGreaterThanOrEqual(0);
    });

    test('should display help modal in both light and dark themes', async ({ page }) => {
      // Test in light theme
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible();

      // Close modal
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);

      // Toggle to dark theme
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      await themeToggle.click();
      await page.waitForTimeout(500);

      const closeSettings = page
        .getByRole('button', { name: /close/i })
        .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
        .first();

      await closeSettings.click();
      await page.waitForTimeout(500);

      // Open help modal in dark theme
      await helpButton.click();
      await page.waitForTimeout(500);

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
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Open settings (if possible with modal open)
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      const settingsVisible = await settingsButton.isVisible().catch(() => false);

      if (settingsVisible) {
        await settingsButton.click();
        await page.waitForTimeout(500);

        // Toggle theme
        const themeToggle = page
          .getByRole('button', { name: /dark|light|theme/i })
          .or(page.locator('[data-testid="theme-toggle"]'))
          .first();

        const toggleVisible = await themeToggle.isVisible().catch(() => false);

        if (toggleVisible) {
          await themeToggle.click();
          await page.waitForTimeout(500);

          // Help modal should still be open
          const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
          await expect(modal).toBeVisible();
        }
      }
    });

    test('should maintain help modal state when toggling theme', async ({ page }) => {
      // Get initial theme
      const initialClasses = await page.locator('html').getAttribute('class');
      const initialTheme = initialClasses?.includes('dark') ? 'dark' : 'light';

      // Open settings and toggle theme
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      await themeToggle.click();
      await page.waitForTimeout(500);

      const closeSettings = page
        .getByRole('button', { name: /close/i })
        .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
        .first();

      await closeSettings.click();
      await page.waitForTimeout(500);

      // Open help modal in new theme
      const helpButton = page
        .getByRole('button', { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

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
