/**
 * shell-one-bar.spec.ts — one top-of-shell pattern (UI-SEED-10, seed#2651).
 *
 * Owner decision 2026-09-15 (fleet): the shell is the left rail plus the page
 * header, and nothing else sits above the page. Seed shipped a second bar —
 * `components/app/HeaderBar.tsx` — which repeated the product mark and the
 * product name beside the rail's own and carried the status, interface, theme
 * and account controls in a row of unlabelled icons.
 *
 * The two assertions are structural rather than visual because that is what
 * the decision is about: where a control lives, not how it looks.
 *
 * - No `<header>` inside `#main-content`. The page notices (`CapabilityWarnings`
 *   and the connection notice) are explicitly allowed by the decision and are
 *   not headers, so this reads exactly the rule.
 * - One product mark on screen. Counting DOM nodes is not enough: the phone
 *   drawer is translated off-canvas rather than unmounted, so the count is over
 *   marks whose box actually intersects the viewport.
 */

import { expect, type Page, test } from '@playwright/test';

import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';

const DESKTOP = { width: 1440, height: 900 };
const PHONE = { width: 390, height: 844 };

/**
 * Every `pageRegistry` route, which is what the acceptance names. The shell is
 * shared, so a subset would very likely pass — but "nothing above the page
 * header on any route" is only proved by every route.
 */
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

test.use({ storageState: AUTH_STORAGE_STATE });

/**
 * Both rails are in the document at once — the phone drawer and the
 * `hidden lg:flex` desktop rail — so every rail locator is scoped to the one
 * actually on screen.
 */
const railControl = (page: Page, testId: string) =>
  page.locator(`[data-testid="${testId}"]:visible`);

/** Marks whose box intersects the viewport — "one product mark per screen". */
async function onScreenMarkCount(page: Page, viewport: { width: number; height: number }) {
  const marks = page.getByTestId('product-mark');
  const boxes = await Promise.all(
    (await marks.all()).map(async (mark) => await mark.boundingBox()),
  );
  return boxes.filter(
    (box) =>
      box !== null &&
      box.x < viewport.width &&
      box.x + box.width > 0 &&
      box.y < viewport.height &&
      box.y + box.height > 0,
  ).length;
}

for (const [name, viewport] of [
  ['desktop 1440x900', DESKTOP],
  ['phone 390x844', PHONE],
] as const) {
  test.describe(`the shell at ${name}`, () => {
    test.beforeEach(async ({ page }) => {
      await disableAnimations(page);
      await page.setViewportSize(viewport);
      await page.goto('/link', { waitUntil: 'domcontentloaded' });
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
    });

    test('renders no bar above the page header, and one product mark, on every route', async ({
      page,
    }) => {
      for (const route of ROUTES) {
        await page.goto(route, { waitUntil: 'domcontentloaded' });
        await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
        await expect(page.locator('#main-content header'), route).toHaveCount(0);
        expect(await onScreenMarkCount(page, viewport), route).toBe(1);
      }
    });
  });
}

test.describe('the rail carries the shell controls', () => {
  test.beforeEach(async ({ page }) => {
    await disableAnimations(page);
    await page.setViewportSize(DESKTOP);
    await page.goto('/link', { waitUntil: 'domcontentloaded' });
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
  });

  test('carries the connection status, and it reads the live state', async ({ page }) => {
    const status = railControl(page, 'rail-status');
    await expect(status).toBeVisible();
    // The rail's dot was hard-coded to the success colour, so it said
    // "connected" on a dead socket. The accessible name has to carry the
    // actual state for that to be fixed rather than merely relocated.
    await expect(status).toHaveAttribute('data-status', /connected|connecting|disconnected|error/);
    // The dot is `aria-hidden` decoration, so the state has to reach a screen
    // reader through the lockup's own name — which is the half of "reachable
    // by keyboard and screen reader" a visibility check cannot see.
    await expect(status).toHaveAccessibleName(/connected|connecting|disconnected|error/i);
  });

  test('carries the theme toggle, and it is keyboard reachable', async ({ page }) => {
    const toggle = railControl(page, 'rail-theme-toggle');
    await expect(toggle).toBeVisible();
    await toggle.focus();
    await expect(toggle).toBeFocused();

    const before = await page.evaluate(() => document.documentElement.classList.contains('dark'));
    await toggle.press('Enter');
    await expect
      .poll(async () => page.evaluate(() => document.documentElement.classList.contains('dark')))
      .toBe(!before);
  });

  test('carries the account menu, with logout inside it', async ({ page }) => {
    const account = railControl(page, 'rail-account');
    await expect(account).toBeVisible();
    await expect(page.getByTestId('rail-logout')).toHaveCount(0);
    await account.click();
    await expect(railControl(page, 'rail-logout')).toBeVisible();
  });

  test('carries the interface selector', async ({ page }) => {
    await expect(railControl(page, 'rail-interface')).toBeVisible();
  });

  test('opens its panels beside the rail when the rail is collapsed', async ({ page }) => {
    await page.getByRole('button', { name: /collapse sidebar/i }).click();
    await railControl(page, 'rail-account').click();

    const logout = railControl(page, 'rail-logout');
    await expect(logout).toBeVisible();

    // The collapsed rail is 64px wide and the panel is anchored to it; a panel
    // that opened to the left would be clipped against the viewport edge.
    const box = await logout.boundingBox();
    expect(box, 'the logout row should have a box').not.toBeNull();
    expect(box?.x ?? -1).toBeGreaterThanOrEqual(64);
  });
});
