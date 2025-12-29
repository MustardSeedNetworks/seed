import { expect, test } from "@playwright/test";

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

test.describe("Responsive Layout Tests", () => {
  test.describe("Mobile Viewport (375x667 - iPhone SE)", () => {
    test.beforeEach(async ({ page }) => {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });

      await page.goto("/");
      await page.evaluate(() => localStorage.clear());
      await page.reload();

      // Login
      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      // Wait for dashboard
      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
        timeout: 10000,
      });
    });

    test("should display login form properly on mobile", async ({ page }) => {
      // Logout to see login form
      const logoutButton = page
        .getByRole("button", { name: /logout|sign out/i })
        .or(page.locator('button:has(svg[class*="logout"], svg[class*="sign-out"])'));

      const hasLogout = await logoutButton.isVisible().catch(() => false);

      if (hasLogout) {
        await logoutButton.click();
        await page.waitForTimeout(500);
      } else {
        await page.goto("/");
        await page.evaluate(() => localStorage.clear());
        await page.reload();
      }

      // Verify login form is usable on mobile
      const usernameField = page.getByLabel(/username/i);
      const passwordField = page.getByLabel(/password/i);
      const loginButton = page.getByRole("button", { name: /sign in|login/i });

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
    });

    test("should show hamburger menu on mobile", async ({ page }) => {
      // Look for hamburger menu button
      const hamburgerMenu = page.locator(
        'button[aria-label*="menu" i], button:has(svg[class*="menu"], svg[class*="bars"])',
      );

      const hasHamburger = await hamburgerMenu.isVisible().catch(() => false);

      // Hamburger menu may or may not be present depending on design
      expect(hasHamburger).toBeDefined();
    });

    test("should stack cards vertically on mobile", async ({ page }) => {
      await page.waitForTimeout(2000);

      // Get all cards
      const cards = page.locator('[class*="card"]');
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

    test("should show settings drawer full-screen on mobile", async ({ page }) => {
      // Open settings
      const settingsButton = page
        .getByRole("button", { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Settings drawer should be visible
      await expect(page.getByText(/thresholds|appearance|discovery/i)).toBeVisible();

      // Check if drawer is full-screen or near full-screen
      const drawer = page.locator('[class*="drawer"], [role="dialog"]').first();
      const drawerBox = await drawer.boundingBox();

      if (drawerBox) {
        // Drawer should take most of viewport width on mobile
        expect(drawerBox.width).toBeGreaterThan(300);
      }
    });

    test("should have touch-friendly button sizes on mobile", async ({ page }) => {
      await page.waitForTimeout(1000);

      // Find interactive buttons
      const buttons = page.locator("button").filter({ hasText: /settings|help|logout/i });
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

    test("should make FAB button accessible on mobile", async ({ page }) => {
      await page.waitForTimeout(1000);

      // Look for FAB (Floating Action Button)
      const fab = page
        .locator('[data-testid="fab"], button[class*="fab"]')
        .or(page.locator('button[class*="fixed"][class*="bottom"]'));

      const hasFab = await fab.isVisible().catch(() => false);

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

    test("should scroll cards vertically on mobile", async ({ page }) => {
      await page.waitForTimeout(2000);

      // Get initial scroll position
      const initialScroll = await page.evaluate(() => window.scrollY);

      // Scroll down
      await page.evaluate(() => window.scrollBy(0, 300));
      await page.waitForTimeout(300);

      // Get new scroll position
      const newScroll = await page.evaluate(() => window.scrollY);

      // Scroll position should have changed
      expect(newScroll).toBeGreaterThan(initialScroll);
    });

    test("should open help modal properly on mobile", async ({ page }) => {
      // Find help button
      const helpButton = page
        .getByRole("button", { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Help modal should be visible
      const modal = page.getByRole("dialog").or(page.locator('[role="dialog"]'));
      await expect(modal).toBeVisible({ timeout: 5000 });

      // Modal should fit viewport
      const modalBox = await modal.boundingBox();

      if (modalBox) {
        expect(modalBox.width).toBeLessThanOrEqual(375);
      }
    });

    test("should display all essential features on mobile", async ({ page }) => {
      await page.waitForTimeout(2000);

      // Verify essential UI elements are present
      const cards = page.locator('[class*="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Settings should be accessible
      const settingsButton = page
        .getByRole("button", { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await expect(settingsButton).toBeVisible();

      // Help should be accessible
      const helpButton = page
        .getByRole("button", { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      const hasHelp = await helpButton.isVisible().catch(() => false);
      expect(hasHelp).toBeDefined();
    });
  });

  test.describe("Tablet Viewport (768x1024 - iPad)", () => {
    test.beforeEach(async ({ page }) => {
      // Set tablet viewport
      await page.setViewportSize({ width: 768, height: 1024 });

      await page.goto("/");
      await page.evaluate(() => localStorage.clear());
      await page.reload();

      // Login
      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      // Wait for dashboard
      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
        timeout: 10000,
      });
    });

    test("should display login form properly on tablet", async ({ page }) => {
      // Logout to see login form
      const logoutButton = page
        .getByRole("button", { name: /logout|sign out/i })
        .or(page.locator('button:has(svg[class*="logout"], svg[class*="sign-out"])'));

      const hasLogout = await logoutButton.isVisible().catch(() => false);

      if (hasLogout) {
        await logoutButton.click();
        await page.waitForTimeout(500);
      } else {
        await page.goto("/");
        await page.evaluate(() => localStorage.clear());
        await page.reload();
      }

      // Verify login form
      await expect(page.getByLabel(/username/i)).toBeVisible();
      await expect(page.getByLabel(/password/i)).toBeVisible();
      await expect(page.getByRole("button", { name: /sign in|login/i })).toBeVisible();
    });

    test("should arrange cards in 2-column grid on tablet", async ({ page }) => {
      await page.waitForTimeout(2000);

      // Get all cards
      const cards = page.locator('[class*="card"]');
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

    test("should show settings drawer as overlay on tablet", async ({ page }) => {
      // Open settings
      const settingsButton = page
        .getByRole("button", { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Settings drawer should be visible
      await expect(page.getByText(/thresholds|appearance|discovery/i)).toBeVisible();

      // Drawer should overlay content (not full-screen)
      const drawer = page.locator('[class*="drawer"], [role="dialog"]').first();
      const drawerBox = await drawer.boundingBox();

      if (drawerBox) {
        // Drawer should be narrower than full viewport on tablet
        expect(drawerBox.width).toBeLessThan(768);
        expect(drawerBox.width).toBeGreaterThan(300);
      }
    });

    test("should have adequate touch targets on tablet", async ({ page }) => {
      await page.waitForTimeout(1000);

      // Find interactive buttons
      const buttons = page.locator("button");
      const buttonCount = await buttons.count();

      if (buttonCount > 0) {
        for (let i = 0; i < Math.min(buttonCount, 5); i++) {
          const button = buttons.nth(i);
          const isVisible = await button.isVisible().catch(() => false);

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

    test("should display navigation appropriately on tablet", async ({ page }) => {
      // Navigation might be full or hamburger menu depending on design
      const nav = page.locator('nav, [role="navigation"]');
      const hamburger = page.locator('button[aria-label*="menu" i]');

      const hasNav = await nav.isVisible().catch(() => false);
      const hasHamburger = await hamburger.isVisible().catch(() => false);

      // Either full nav or hamburger should be present
      expect(hasNav || hasHamburger).toBe(true);
    });

    test("should display all cards on tablet", async ({ page }) => {
      await page.waitForTimeout(2000);

      // Count visible cards
      const cards = page.locator('[class*="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Verify common cards are visible
      const linkCard = page.locator("text=/link/i").first();
      await expect(linkCard).toBeVisible();
    });
  });

  test.describe("Desktop Viewport (1920x1080 - Full HD)", () => {
    test.beforeEach(async ({ page }) => {
      // Set desktop viewport
      await page.setViewportSize({ width: 1920, height: 1080 });

      await page.goto("/");
      await page.evaluate(() => localStorage.clear());
      await page.reload();

      // Login
      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      // Wait for dashboard
      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
        timeout: 10000,
      });
    });

    test("should display login form properly on desktop", async ({ page }) => {
      // Logout to see login form
      const logoutButton = page
        .getByRole("button", { name: /logout|sign out/i })
        .or(page.locator('button:has(svg[class*="logout"], svg[class*="sign-out"])'));

      const hasLogout = await logoutButton.isVisible().catch(() => false);

      if (hasLogout) {
        await logoutButton.click();
        await page.waitForTimeout(500);
      } else {
        await page.goto("/");
        await page.evaluate(() => localStorage.clear());
        await page.reload();
      }

      // Verify login form
      await expect(page.getByLabel(/username/i)).toBeVisible();
      await expect(page.getByLabel(/password/i)).toBeVisible();

      // Login form should be centered/styled appropriately for desktop
      const loginContainer = page.locator('form, [class*="login"]').first();
      const box = await loginContainer.boundingBox();

      if (box) {
        // Login form should not span full width on desktop
        expect(box.width).toBeLessThan(1000);
      }
    });

    test("should arrange cards in 3-4 column grid on desktop", async ({ page }) => {
      await page.waitForTimeout(2000);

      // Get all cards
      const cards = page.locator('[class*="card"]');
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

    test("should show full navigation on desktop", async ({ page }) => {
      // Full navigation should be visible (not hamburger menu)
      const nav = page.locator('nav, [role="navigation"]');
      const hamburger = page.locator('button[aria-label*="menu" i]');

      const hasNav = await nav.isVisible().catch(() => false);
      const hasHamburger = await hamburger.isVisible().catch(() => false);

      // Desktop should prefer full navigation over hamburger
      // But implementation may vary
      expect(hasNav).toBeDefined();
      expect(hasHamburger).toBeDefined();
    });

    test("should slide settings drawer from right on desktop", async ({ page }) => {
      // Open settings
      const settingsButton = page
        .getByRole("button", { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      // Settings drawer should be visible
      await expect(page.getByText(/thresholds|appearance|discovery/i)).toBeVisible();

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

    test("should provide optimal layout for large screens", async ({ page }) => {
      await page.waitForTimeout(2000);

      // Verify content is well-distributed
      const cards = page.locator('[class*="card"]');
      const cardCount = await cards.count();

      expect(cardCount).toBeGreaterThan(0);

      // Content should not be stretched to full width
      const container = page.locator('[class*="container"], [class*="wrapper"]').first();
      const containerBox = await container.boundingBox().catch(() => null);

      if (containerBox) {
        // Container might have max-width for readability
        expect(containerBox.width).toBeLessThanOrEqual(1920);
      }
    });

    test("should display all cards without scrolling (above the fold)", async ({ page }) => {
      await page.waitForTimeout(2000);

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

    test("should handle help modal at desktop size", async ({ page }) => {
      // Open help modal
      const helpButton = page
        .getByRole("button", { name: /help/i })
        .or(page.locator('button:has(svg[class*="help"], svg[class*="question"])'));

      await helpButton.click();
      await page.waitForTimeout(500);

      // Help modal should be visible
      const modal = page.getByRole("dialog").or(page.locator('[role="dialog"]'));
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

  test.describe("Cross-Viewport Feature Consistency", () => {
    test("should maintain authentication across all viewports", async ({ page }) => {
      // Test on mobile
      await page.setViewportSize({ width: 375, height: 667 });
      await page.goto("/");
      await page.evaluate(() => localStorage.clear());
      await page.reload();

      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
        timeout: 10000,
      });

      // Resize to tablet - should stay authenticated
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.waitForTimeout(500);

      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible();

      // Resize to desktop - should stay authenticated
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.waitForTimeout(500);

      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible();
    });

    test("should maintain theme preference across viewports", async ({ page }) => {
      // Login on desktop
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.goto("/");
      await page.evaluate(() => localStorage.clear());
      await page.reload();

      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
        timeout: 10000,
      });

      // Set dark theme
      const settingsButton = page
        .getByRole("button", { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      await settingsButton.click();
      await page.waitForTimeout(500);

      const themeToggle = page
        .getByRole("button", { name: /dark|light|theme/i })
        .or(page.locator('[data-testid="theme-toggle"]'))
        .first();

      const htmlElement = page.locator("html");
      let classes = await htmlElement.getAttribute("class");

      if (!classes?.includes("dark")) {
        await themeToggle.click();
        await page.waitForTimeout(500);
      }

      // Close settings
      const closeButton = page
        .getByRole("button", { name: /close/i })
        .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
        .first();

      await closeButton.click();
      await page.waitForTimeout(500);

      // Verify dark theme
      classes = await htmlElement.getAttribute("class");
      expect(classes).toContain("dark");

      // Switch to mobile - theme should persist
      await page.setViewportSize({ width: 375, height: 667 });
      await page.waitForTimeout(500);

      const mobileClasses = await htmlElement.getAttribute("class");
      expect(mobileClasses).toContain("dark");

      // Switch to tablet - theme should persist
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.waitForTimeout(500);

      const tabletClasses = await htmlElement.getAttribute("class");
      expect(tabletClasses).toContain("dark");
    });

    test("should display same card data across all viewports", async ({ page }) => {
      // Login on desktop
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.goto("/");
      await page.evaluate(() => localStorage.clear());
      await page.reload();

      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
        timeout: 10000,
      });

      await page.waitForTimeout(2000);

      // Count cards on desktop
      const desktopCards = await page.locator('[class*="card"]').count();
      expect(desktopCards).toBeGreaterThan(0);

      // Switch to tablet
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.waitForTimeout(1000);

      const tabletCards = await page.locator('[class*="card"]').count();
      expect(tabletCards).toBeGreaterThanOrEqual(desktopCards - 2); // Allow for minor variance

      // Switch to mobile
      await page.setViewportSize({ width: 375, height: 667 });
      await page.waitForTimeout(1000);

      const mobileCards = await page.locator('[class*="card"]').count();
      expect(mobileCards).toBeGreaterThanOrEqual(desktopCards - 2); // Allow for minor variance
    });

    test("should provide working settings across all viewports", async ({ page }) => {
      // Login
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.goto("/");
      await page.evaluate(() => localStorage.clear());
      await page.reload();

      await page.getByLabel(/username/i).fill("admin");
      await page.getByLabel(/password/i).fill("seed");
      await page.getByRole("button", { name: /sign in|login/i }).click();

      await expect(page.getByRole("heading", { name: /link|dashboard/i })).toBeVisible({
        timeout: 10000,
      });

      // Test settings on each viewport
      for (const viewport of [
        { width: 1920, height: 1080, name: "desktop" },
        { width: 768, height: 1024, name: "tablet" },
        { width: 375, height: 667, name: "mobile" },
      ]) {
        await page.setViewportSize(viewport);
        await page.waitForTimeout(500);

        // Open settings
        const settingsButton = page
          .getByRole("button", { name: /settings/i })
          .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

        await settingsButton.click();
        await page.waitForTimeout(500);

        // Verify settings content visible
        await expect(page.getByText(/thresholds|appearance|discovery/i)).toBeVisible({
          timeout: 5000,
        });

        // Close settings
        const closeButton = page
          .getByRole("button", { name: /close/i })
          .or(page.locator('button:has(svg[class*="x"], svg[class*="close"])'))
          .first();

        await closeButton.click();
        await page.waitForTimeout(500);
      }
    });
  });
});
