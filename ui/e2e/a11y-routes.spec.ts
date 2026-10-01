/**
 * a11y-routes.spec.ts — every page passes axe in a real browser (#388).
 *
 * The Storybook run gates each story, but a component with no story is
 * checked by nothing: 44 interactive components — the settings sections, most
 * cards, the sidebar, the page header, the login form — have none. Walking
 * every route in `pageRegistry` reaches them as the operator does, assembled,
 * in both engines and at both widths, which also catches what a story cannot:
 * duplicate ids and landmark structure across a whole page.
 *
 * color-contrast is off for the reason `help-closeout.spec.ts` gives: it
 * belongs to the theme row (UI-SEED-7, seed#2647) with its own acceptance, and
 * webkit and chromium disagree about the rail gradient.
 */
import { expect, type Page, test } from '@playwright/test';
import axe from 'axe-core';

import { AUTH_STORAGE_STATE, disableAnimations, sidebarSettingsButton } from './helpers/auth';

const ROUTES = [
  '/link',
  '/network',
  '/path',
  '/wifi',
  '/security',
  '/performance',
  '/reports',
  '/logs',
  '/polling-targets',
  '/topology',
  '/alerts',
];

declare global {
  interface Window {
    /** Installed by the init script below from `axe.source`. */
    axe: typeof axe;
  }
}

type Violation = { id: string; impact: string | null; nodes: string[] };

async function axeViolations(page: Page): Promise<Violation[]> {
  const violations = await page.evaluate(async () => {
    const report = await window.axe.run(document, {
      runOnly: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'],
      rules: { 'color-contrast': { enabled: false } },
    });
    return report.violations.map(({ id, impact, nodes }) => ({
      id,
      impact: impact ?? null,
      nodes: nodes.map(({ html }) => html.slice(0, 200)),
    }));
  });
  await test.info().attach('axe-violations.json', {
    body: JSON.stringify(violations, null, 2),
    contentType: 'application/json',
  });
  return violations;
}

test.use({ storageState: AUTH_STORAGE_STATE });

test.beforeEach(async ({ page }) => {
  await disableAnimations(page);
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.addInitScript({ content: axe.source });
});

for (const width of [1440, 390]) {
  test.describe(`${width}px`, () => {
    test.beforeEach(async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
    });

    for (const route of ROUTES) {
      test(`${route} has no axe violations`, async ({ page }) => {
        await page.goto(route);
        await expect(page.getByTestId('page-header-title')).toBeVisible();
        await page.waitForLoadState('networkidle');
        expect(await axeViolations(page)).toEqual([]);
      });
    }

    test('the open settings drawer has no axe violations', async ({ page }) => {
      await page.goto('/link');
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      if (width < 1024) await page.getByTestId('mobile-menu-toggle').click();
      await sidebarSettingsButton(page).click();
      const drawer = page.getByTestId('settings-drawer');
      await expect(drawer).toBeVisible();
      // Sections render their body only when expanded, and most of the
      // story-less components are those bodies. Expanding one can reveal a
      // nested collapsible, so repeat until nothing is left closed.
      const collapsed = drawer.locator('[aria-expanded="false"]');
      for (let opened = 0; opened < 100 && (await collapsed.count()) > 0; opened++) {
        await collapsed.first().click();
      }
      await expect(collapsed).toHaveCount(0);
      await page.waitForLoadState('networkidle');
      expect(await axeViolations(page)).toEqual([]);
    });

    test('the open help drawer has no axe violations', async ({ page }) => {
      await page.goto('/network');
      await page.getByTestId('page-header-help-button').click();
      await expect(page.getByTestId('help-drawer')).toBeVisible();
      expect(await axeViolations(page)).toEqual([]);
    });
  });
}

test.describe('signed out', () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  for (const width of [1440, 390]) {
    test(`${width}px the sign-in page has no axe violations`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto('/');
      await expect(page.getByTestId('login-title')).toBeVisible();
      expect(await axeViolations(page)).toEqual([]);
    });
  }
});

/**
 * axe cannot see focus. These are the two keyboard properties it misses for
 * the shell's three overlays: Tab never leaves an open dialog, and Escape
 * closes it and hands focus back to the control that opened it, so a
 * keyboard operator is not dropped at the top of the page.
 */
test.describe('overlays keep and return keyboard focus', () => {
  const overlays = [
    {
      name: 'settings drawer',
      open: (page: Page) => sidebarSettingsButton(page),
      dialog: (page: Page) => page.getByTestId('settings-drawer'),
    },
    {
      name: 'help drawer',
      open: (page: Page) => page.getByTestId('page-header-help-button'),
      dialog: (page: Page) => page.getByTestId('help-drawer'),
    },
  ];

  for (const { name, open, dialog } of overlays) {
    test(`the ${name} traps Tab and restores focus on Escape`, async ({ page }) => {
      await page.setViewportSize({ width: 1440, height: 900 });
      await page.goto('/network');
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      const trigger = open(page);
      await trigger.focus();
      await page.keyboard.press('Enter');
      await expect(dialog(page)).toBeVisible();

      for (const key of ['Tab', 'Shift+Tab']) {
        for (let presses = 0; presses < 40; presses++) {
          await page.keyboard.press(key);
          expect(
            await dialog(page).evaluate((root) => root.contains(document.activeElement)),
            `${key} #${presses + 1} left the ${name}`,
          ).toBe(true);
        }
      }

      await page.keyboard.press('Escape');
      await expect(dialog(page)).toBeHidden();
      await expect(trigger).toBeFocused();
    });
  }

  test('the command palette traps Tab and closes on Escape', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/network');
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await page.keyboard.press('ControlOrMeta+k');
    // The Radix dialog node has no box of its own (its children are fixed),
    // so Playwright calls it hidden; its input is what the operator sees.
    const palette = page.getByRole('dialog');
    await expect(palette.getByRole('combobox')).toBeVisible();
    expect(await axeViolations(page)).toEqual([]);

    for (let presses = 0; presses < 20; presses++) {
      await page.keyboard.press('Tab');
      expect(
        await palette.evaluate((root) => root.contains(document.activeElement)),
        `Tab #${presses + 1} left the command palette`,
      ).toBe(true);
    }

    await page.keyboard.press('Escape');
    await expect(palette).toHaveCount(0);
  });
});
