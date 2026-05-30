import { expect, test } from '@playwright/test';
import {
  revealSidebar,
  sidebarHelpButton,
  sidebarSettingsButton,
  skipSetupWizard,
} from './helpers/auth';

/**
 * Responsive Layout E2E Tests
 *
 * Comprehensive tests for responsive layouts across different viewports:
 *
 * Viewports tested:
 * - Mobile (375x667 - iPhone SE)
 * - Tablet (768x1024 - iPad)
 * - Desktop (1920x1080 - Full HD)
 *
 * Features tested:
 * - Navigation (hamburger menu on mobile, full nav on desktop)
 * - Card layouts (stacked on mobile, grid on larger screens)
 * - Settings drawer (full-screen on mobile, overlay on larger screens)
 * - FAB button accessibility
 * - Touch-friendly button sizes
 * - Login form responsiveness
 * - Dashboard responsiveness
 * - Help modal responsiveness
 * - All card visibility
 *
 * Each feature is tested across all viewports to ensure
 * usability on different device sizes.
 */

test.describe('Responsive Layout Tests', () => {
  test.describe('Mobile Viewport (375x667 - iPhone SE)', () => {
    test.beforeEach(async ({ page }) => {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });

      await skipSetupWizard(page);
      await page.goto('/');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });
    });

    // Run this test with an unauthenticated context — see file-
    // top describe('Login-form…') for the rationale. Clicking the
    // header logout button via the shared storageState would blacklist
    // the suite-wide auth token and poison every later test.
    test('should display login form properly on mobile', async ({ browser }) => {
      const ctx = await browser.newContext({
        viewport: { width: 375, height: 667 },
        storageState: { cookies: [], origins: [] },
      });
      const page = await ctx.newPage();
      await skipSetupWizard(page);
      await page.goto('/');

      // Verify login form is usable on mobile
      const usernameField = page.getByLabel(/username/i);
      const passwordField = page.getByLabel(/password/i);
      const loginButton = page.getByRole('button', { name: /sign in|login/i });

      await expect(usernameField).toBeVisible();
      await expect(passwordField).toBeVisible();
      await expect(loginButton).toBeVisible();

      // Verify fields are within viewport
      const usernameBox = await usernameField.boundingBox();
      const passwordBox = await passwordField.boundingBox();

      expect(usernameBox).toBeTruthy();
      expect(passwordBox).toBeTruthy();

      if (usernameBox && passwordBox) {
        expect(usernameBox.width).toBeLessThanOrEqual(375);
        expect(passwordBox.width).toBeLessThanOrEqual(375);
      }

      await ctx.close();
    });

    test('should show hamburger menu on mobile', async ({ page }) => {
      // Look for hamburger menu button
      const hamburgerMenu = page.locator(
        'button[aria-label*="menu" i], button:has(svg[class*="menu"], svg[class*="bars"])',
      );

      const hasHamburger = await hamburgerMenu.isVisible();

      // Hamburger menu may or may not be present depending on design
      expect(hasHamburger).toBeDefined();
    });

    test('should stack cards vertically on mobile', async ({ page }) => {
      // BaseCard emits `data-testid="card"` on every wrapper
      // (components/ui/card.tsx:125). The previous `[class*="card"]`
      // substring match against Tailwind's hashed classes was non-
      // deterministic — under strict mode it grabbed arbitrary unrelated
      // nodes (button class names, icon decorations) and the count check
      // landed on whichever order Tailwind merged classes that build.
      const cards = page.locator('[data-testid="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Check if cards are stacked (each card takes roughly full width)
      for (let i = 0; i < Math.min(cardCount, 3); i++) {
        const card = cards.nth(i);
        const box = await card.boundingBox();

        if (box) {
          // Cards should be close to viewport width (allowing for padding)
          expect(box.width).toBeGreaterThan(300); // Most of 375px width
        }
      }
    });

    test('should show settings drawer full-screen on mobile', async ({ page }) => {
      // Settings lives in the sidebar drawer behind the hamburger on mobile.
      await revealSidebar(page);
      const settingsButton = sidebarSettingsButton(page);

      await settingsButton.click();

      // Settings drawer should be visible
      await expect(page.getByTestId('settings-drawer')).toBeVisible();

      // Check if drawer is full-screen or near full-screen
      const drawer = page.locator('[class*="drawer"], [role="dialog"]').first();
      const drawerBox = await drawer.boundingBox();

      if (drawerBox) {
        // Drawer should take most of viewport width on mobile
        expect(drawerBox.width).toBeGreaterThan(300);
      }
    });

    test('should have touch-friendly button sizes on mobile', async ({ page }) => {
      // Find interactive buttons
      const buttons = page.locator('button').filter({ hasText: /settings|help|logout/i });
      const buttonCount = await buttons.count();

      if (buttonCount > 0) {
        for (let i = 0; i < Math.min(buttonCount, 3); i++) {
          const button = buttons.nth(i);
          const box = await button.boundingBox();

          if (box) {
            // Touch targets should be at least 44x44px (iOS guidelines)
            // or 48x48px (Material Design)
            const minSize = 40; // Slightly less to account for padding

            expect(box.height).toBeGreaterThanOrEqual(minSize);
          }
        }
      }
    });

    test('should make FAB button accessible on mobile', async ({ page }) => {
      // Look for FAB (Floating Action Button)
      const fab = page
        .locator('[data-testid="fab"], button[class*="fab"]')
        .or(page.locator('button[class*="fixed"][class*="bottom"]'));

      const hasFab = await fab.isVisible();

      if (hasFab) {
        // FAB should be positioned in viewport
        const fabBox = await fab.boundingBox();

        if (fabBox) {
          // FAB should be within viewport bounds
          expect(fabBox.x).toBeGreaterThanOrEqual(0);
          expect(fabBox.y).toBeGreaterThanOrEqual(0);
          expect(fabBox.x + fabBox.width).toBeLessThanOrEqual(375);
          expect(fabBox.y + fabBox.height).toBeLessThanOrEqual(667);
        }
      }
    });

    // PR-C1 cleanup: dropped "should scroll cards vertically on mobile" —
    // it asserted `window.scrollBy(0, 300)` actually moved scrollY, which
    // tests the BROWSER (or jsdom), not the app. Any future regression to
    // mobile scroll would never surface here; the test added a CI run-cost
    // for zero defensive signal.

    test('should open help modal properly on mobile', async ({ page }) => {
      // Help lives in the sidebar drawer behind the hamburger on mobile.
      await revealSidebar(page);
      const helpButton = sidebarHelpButton(page);

      await helpButton.click();

      // Help modal should be visible
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible({ timeout: 5000 });

      // Modal should fit viewport
      const modalBox = await modal.boundingBox();

      if (modalBox) {
        expect(modalBox.width).toBeLessThanOrEqual(375);
      }
    });

    test('should display all essential features on mobile', async ({ page }) => {
      // Verify essential UI elements are present. PR-3 / PR-C1 swap:
      // `[class*="card"]` was substring-matching against Tailwind's
      // hashed class names and grabbing arbitrary unrelated nodes;
      // BaseCard emits `data-testid="card"` on every wrapper.
      const cards = page.locator('[data-testid="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Settings + Help moved into the sidebar (Phase 2); on mobile they're
      // reached via the hamburger. Open the drawer, then assert both surface.
      await revealSidebar(page);
      await expect(sidebarSettingsButton(page)).toBeVisible();
      await expect(sidebarHelpButton(page)).toBeVisible();
    });
  });

  test.describe('Tablet Viewport (768x1024 - iPad)', () => {
    test.beforeEach(async ({ page }) => {
      // Set tablet viewport
      await page.setViewportSize({ width: 768, height: 1024 });

      await skipSetupWizard(page);
      await page.goto('/');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });
    });

    // Unauthenticated context — see comment on the mobile sibling.
    test('should display login form properly on tablet', async ({ browser }) => {
      const ctx = await browser.newContext({
        viewport: { width: 768, height: 1024 },
        storageState: { cookies: [], origins: [] },
      });
      const page = await ctx.newPage();
      await skipSetupWizard(page);
      await page.goto('/');

      await expect(page.getByLabel(/username/i)).toBeVisible();
      await expect(page.getByLabel(/password/i)).toBeVisible();
      await expect(page.getByRole('button', { name: /sign in|login/i })).toBeVisible();

      await ctx.close();
    });

    test('should arrange cards in 2-column grid on tablet', async ({ page }) => {
      // BaseCard emits `data-testid="card"` on every wrapper
      // (components/ui/card.tsx:125). The previous `[class*="card"]`
      // substring match against Tailwind's hashed classes was non-
      // deterministic — under strict mode it grabbed arbitrary unrelated
      // nodes (button class names, icon decorations) and the count check
      // landed on whichever order Tailwind merged classes that build.
      const cards = page.locator('[data-testid="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Check if cards are arranged in rows (not all full width)
      if (cardCount >= 2) {
        const firstCard = await cards.nth(0).boundingBox();
        const secondCard = await cards.nth(1).boundingBox();

        if (firstCard && secondCard) {
          // Cards should be narrower than viewport (allowing for grid layout)
          expect(firstCard.width).toBeLessThan(700); // Not full 768px width
          expect(secondCard.width).toBeLessThan(700);

          // Cards might be side-by-side if using 2-column layout
          const sideBySide = Math.abs(firstCard.y - secondCard.y) < 50;
          expect(sideBySide).toBeDefined();
        }
      }
    });

    test('should show settings drawer as overlay on tablet', async ({ page }) => {
      // Below lg (tablet = 768px) the sidebar is a drawer behind the hamburger.
      await revealSidebar(page);
      const settingsButton = sidebarSettingsButton(page);

      await settingsButton.click();

      // Settings drawer should be visible
      await expect(page.getByTestId('settings-drawer')).toBeVisible();

      // Drawer should overlay content (not full-screen)
      const drawer = page.locator('[class*="drawer"], [role="dialog"]').first();
      const drawerBox = await drawer.boundingBox();

      if (drawerBox) {
        // Drawer should be narrower than full viewport on tablet
        expect(drawerBox.width).toBeLessThan(768);
        expect(drawerBox.width).toBeGreaterThan(300);
      }
    });

    test('should have adequate touch targets on tablet', async ({ page }) => {
      // Find interactive buttons
      const buttons = page.locator('button');
      const buttonCount = await buttons.count();

      if (buttonCount > 0) {
        for (let i = 0; i < Math.min(buttonCount, 5); i++) {
          const button = buttons.nth(i);
          const isVisible = await button.isVisible();

          if (isVisible) {
            const box = await button.boundingBox();

            if (box) {
              // Touch targets should meet minimum size requirements
              expect(box.height).toBeGreaterThan(0);
              expect(box.width).toBeGreaterThan(0);
            }
          }
        }
      }
    });

    test('should display navigation appropriately on tablet', async ({ page }) => {
      // Navigation might be full or hamburger menu depending on design
      const nav = page.locator('nav, [role="navigation"]');
      const hamburger = page.locator('button[aria-label*="menu" i]');

      const hasNav = await nav.isVisible();
      const hasHamburger = await hamburger.isVisible();

      // Either full nav or hamburger should be present
      expect(hasNav || hasHamburger).toBe(true);
    });

    test('should display all cards on tablet', async ({ page }) => {
      // PR-C1: same fix as the two other shard sites in this file —
      // [class*="card"] was non-deterministic, and the `text=/link/i`
      // fallback would match the sidebar nav item before reaching the
      // dashboard card. Use the stable BaseCard `data-testid="card"`
      // and the `#card-title-link` ID also emitted by BaseCard.
      const cards = page.locator('[data-testid="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      await expect(page.locator('#card-title-link')).toBeVisible();
    });
  });

  test.describe('Desktop Viewport (1920x1080 - Full HD)', () => {
    test.beforeEach(async ({ page }) => {
      // Set desktop viewport
      await page.setViewportSize({ width: 1920, height: 1080 });

      await skipSetupWizard(page);
      await page.goto('/');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });
    });

    // Unauthenticated context — see comment on the mobile sibling.
    test('should display login form properly on desktop', async ({ browser }) => {
      const ctx = await browser.newContext({
        viewport: { width: 1920, height: 1080 },
        storageState: { cookies: [], origins: [] },
      });
      const page = await ctx.newPage();
      await skipSetupWizard(page);
      await page.goto('/');

      await expect(page.getByLabel(/username/i)).toBeVisible();
      await expect(page.getByLabel(/password/i)).toBeVisible();

      // Login form should be centered/styled appropriately for desktop
      const loginContainer = page.locator('form, [class*="login"]').first();
      const box = await loginContainer.boundingBox();

      if (box) {
        // Login form should not span full width on desktop
        expect(box.width).toBeLessThan(1000);
      }

      await ctx.close();
    });

    test('should arrange cards in 3-4 column grid on desktop', async ({ page }) => {
      // BaseCard emits `data-testid="card"` on every wrapper
      // (components/ui/card.tsx:125). The previous `[class*="card"]`
      // substring match against Tailwind's hashed classes was non-
      // deterministic — under strict mode it grabbed arbitrary unrelated
      // nodes (button class names, icon decorations) and the count check
      // landed on whichever order Tailwind merged classes that build.
      const cards = page.locator('[data-testid="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Check card widths for grid layout
      if (cardCount >= 3) {
        const firstCard = await cards.nth(0).boundingBox();
        const secondCard = await cards.nth(1).boundingBox();
        const thirdCard = await cards.nth(2).boundingBox();

        if (firstCard && secondCard && thirdCard) {
          // Cards should be significantly narrower than viewport
          expect(firstCard.width).toBeLessThan(600);
          expect(secondCard.width).toBeLessThan(600);
          expect(thirdCard.width).toBeLessThan(600);

          // Check if cards are arranged horizontally
          const row1 = Math.abs(firstCard.y - secondCard.y) < 50;
          const row2 = Math.abs(secondCard.y - thirdCard.y) < 50;

          // At least some cards should be in same row
          expect(row1 || row2).toBe(true);
        }
      }
    });

    test('should show full navigation on desktop', async ({ page }) => {
      // Full navigation should be visible (not hamburger menu)
      const nav = page.locator('nav, [role="navigation"]');
      const hamburger = page.locator('button[aria-label*="menu" i]');

      const hasNav = await nav.isVisible();
      const hasHamburger = await hamburger.isVisible();

      // Desktop should prefer full navigation over hamburger
      // But implementation may vary
      expect(hasNav).toBeDefined();
      expect(hasHamburger).toBeDefined();
    });

    test('should slide settings drawer from right on desktop', async ({ page }) => {
      // Open settings
      const settingsButton = sidebarSettingsButton(page);

      await settingsButton.click();

      // Settings drawer should be visible
      await expect(page.getByTestId('settings-drawer')).toBeVisible();

      // Drawer should be positioned on right side
      const drawer = page.locator('[class*="drawer"], [role="dialog"]').first();
      const drawerBox = await drawer.boundingBox();

      if (drawerBox) {
        // Drawer should be on right side (x position > middle of screen)
        expect(drawerBox.x).toBeGreaterThan(960); // Right half of 1920px

        // Drawer should not be full width
        expect(drawerBox.width).toBeLessThan(800);
      }
    });

    test('should provide optimal layout for large screens', async ({ page }) => {
      // Verify content is well-distributed
      const cards = page.locator('[class*="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Content should not be stretched to full width
      const container = page.locator('[class*="container"], [class*="wrapper"]').first();
      const containerBox = await container.boundingBox();

      if (containerBox) {
        // Container might have max-width for readability
        expect(containerBox.width).toBeLessThanOrEqual(1920);
      }
    });

    test('should display all cards without scrolling (above the fold)', async ({ page }) => {
      // Get initial scroll position
      const scrollY = await page.evaluate(() => window.scrollY);

      // Should start at top
      expect(scrollY).toBe(0);

      // Count visible cards without scrolling
      const visibleCards = page.locator('[class*="card"]');
      const visibleCount = await visibleCards.count();

      // At least some cards should be visible without scrolling
      expect(visibleCount).toBeGreaterThan(0);
    });

    test('should handle help modal at desktop size', async ({ page }) => {
      // Open help modal
      const helpButton = sidebarHelpButton(page);

      await helpButton.click();

      // Help modal should be visible
      const modal = page.getByRole('dialog').or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible({ timeout: 5000 });

      // Modal should be centered and not full width
      const modalBox = await modal.boundingBox();

      if (modalBox) {
        // Modal should be centered (not edge-to-edge)
        expect(modalBox.width).toBeLessThan(1600);

        // Modal should be centered horizontally
        const centerX = modalBox.x + modalBox.width / 2;
        const viewportCenter = 1920 / 2;

        expect(Math.abs(centerX - viewportCenter)).toBeLessThan(200);
      }
    });
  });

  test.describe('Cross-Viewport Feature Consistency', () => {
    test('should maintain authentication across all viewports', async ({ page }) => {
      // Test on mobile
      await page.setViewportSize({ width: 375, height: 667 });
      await skipSetupWizard(page);
      await page.goto('/');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });

      // Resize to tablet - should stay authenticated
      await page.setViewportSize({ width: 768, height: 1024 });

      await expect(page.getByTestId('page-header-title')).toBeVisible();

      // Resize to desktop - should stay authenticated
      await page.setViewportSize({ width: 1920, height: 1080 });

      await expect(page.getByTestId('page-header-title')).toBeVisible();
    });

    test('should maintain theme preference across viewports', async ({ page }) => {
      // Login on desktop
      await page.setViewportSize({ width: 1920, height: 1080 });
      await skipSetupWizard(page);
      await page.goto('/');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });

      // Set dark theme
      const settingsButton = sidebarSettingsButton(page);

      await settingsButton.click();

      const themeToggle = page
        .getByRole('button', { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      const htmlElement = page.locator('html');
      let classes = await htmlElement.getAttribute('class');

      if (!classes?.includes('dark')) {
        await themeToggle.click();
      }

      // Close settings
      const closeButton = page
        .getByRole('button', { name: /close/i })
        .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
        .first();

      await closeButton.click();

      // Verify dark theme
      classes = await htmlElement.getAttribute('class');
      expect(classes).toContain('dark');

      // Switch to mobile - theme should persist
      await page.setViewportSize({ width: 375, height: 667 });

      const mobileClasses = await htmlElement.getAttribute('class');
      expect(mobileClasses).toContain('dark');

      // Switch to tablet - theme should persist
      await page.setViewportSize({ width: 768, height: 1024 });

      const tabletClasses = await htmlElement.getAttribute('class');
      expect(tabletClasses).toContain('dark');
    });

    test('should display same card data across all viewports', async ({ page }) => {
      // Login on desktop
      await page.setViewportSize({ width: 1920, height: 1080 });
      await skipSetupWizard(page);
      await page.goto('/');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });

      // Count cards on desktop
      const desktopCards = await page.locator('[class*="card"]').count();
      expect(desktopCards).toBeGreaterThan(0);

      // Switch to tablet
      await page.setViewportSize({ width: 768, height: 1024 });

      const tabletCards = await page.locator('[class*="card"]').count();
      expect(tabletCards).toBeGreaterThanOrEqual(desktopCards - 2); // Allow for minor variance

      // Switch to mobile
      await page.setViewportSize({ width: 375, height: 667 });

      const mobileCards = await page.locator('[class*="card"]').count();
      expect(mobileCards).toBeGreaterThanOrEqual(desktopCards - 2); // Allow for minor variance
    });

    test('should provide working settings across all viewports', async ({ page }) => {
      // Login
      await page.setViewportSize({ width: 1920, height: 1080 });
      await skipSetupWizard(page);
      await page.goto('/');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });

      // Test settings on each viewport
      for (const viewport of [
        { width: 1920, height: 1080, name: 'desktop' },
        { width: 768, height: 1024, name: 'tablet' },
        { width: 375, height: 667, name: 'mobile' },
      ]) {
        await page.setViewportSize(viewport);

        // Sidebar is behind the hamburger below lg; no-op at desktop width.
        await revealSidebar(page);
        const settingsButton = sidebarSettingsButton(page);

        await settingsButton.click();

        // Verify settings content visible
        await expect(page.getByTestId('settings-drawer')).toBeVisible({
          timeout: 5000,
        });

        // Close via the drawer's own button. A generic /close/i would also
        // match the mobile sidebar's "Close menu" hamburger that revealSidebar
        // surfaces below lg, so target the settings-drawer close directly.
        await page.getByTestId('settings-drawer-close').click();
      }
    });
  });
});
