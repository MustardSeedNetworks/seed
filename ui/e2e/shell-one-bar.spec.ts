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
import { expectFocused } from './helpers/focus';

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
  '/device-groups',
  '/ipam',
  '/lifecycle',
  '/ip-sla',
  '/topology',
  '/interfaces',
  '/device-configs',
  '/flows',
  '/alerts',
  '/dashboard',
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
    });

    // A test per route rather than one loop: eleven webkit navigations do not
    // fit the suite's 30s per-test budget, and a per-route name says which
    // route broke.
    for (const route of ROUTES) {
      test(`${route} renders no bar above the page header, and one product mark`, async ({
        page,
      }) => {
        await page.goto(route, { waitUntil: 'domcontentloaded' });
        await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
        // The shared PageHeader is itself a <header>; any other one is a bar above it.
        await expect(
          page.locator('#main-content header:not([data-testid="page-header"])'),
        ).toHaveCount(0);
        expect(await onScreenMarkCount(page, viewport)).toBe(1);
      });
    }
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
    // The state has to reach the operator in words, not only as a colour.
    await expect(status).toContainText(/connected|connecting|disconnected|error/i);
  });

  test('carries the theme toggle, and it is keyboard reachable', async ({ page }) => {
    const toggle = railControl(page, 'rail-theme');
    await expect(toggle).toBeVisible();
    await toggle.focus();
    await expectFocused(toggle);

    const before = await page.evaluate(() => document.documentElement.classList.contains('dark'));
    await toggle.press('Enter');
    await expect
      .poll(async () => page.evaluate(() => document.documentElement.classList.contains('dark')))
      .toBe(!before);
  });

  test('carries sign out, Help and Settings as icon buttons', async ({ page }) => {
    for (const id of ['rail-sign-out', 'rail-help', 'rail-settings']) {
      await expect(railControl(page, id)).toBeVisible();
    }
  });

  test('carries the interface switch as its one product control', async ({ page }) => {
    const slot = railControl(page, 'rail-product-slot');
    await expect(slot).toBeVisible();
    await slot.click();
    await expect(page.getByRole('menu')).toBeVisible();
    await expect(page.getByRole('menuitemradio', { name: /Wi-?Fi/ })).toBeVisible();
  });

  test('opens its menu beside the rail when the rail is collapsed', async ({ page }) => {
    await railControl(page, 'rail-collapse').click();
    await railControl(page, 'rail-product-slot').click();

    const item = page.getByRole('menuitemradio').first();
    await expect(item).toBeVisible();

    // The collapsed rail is 64px wide and the menu is anchored beside it; one
    // that opened over the rail would hide the controls it hangs off.
    const box = await item.boundingBox();
    expect(box, 'the menu item should have a box').not.toBeNull();
    expect(box?.x ?? -1).toBeGreaterThanOrEqual(64);
  });
});
