/**
 * fab-phone-clearance.spec.ts — the run control must not sit on the numbers.
 *
 * At 390×844 the fixed "Run all tests" FAB was pinned `bottom-20 right-6`,
 * which put its 56×56 box on top of the status band's figures on `/link`
 * (#2646; the UI audit's `light/mobile/link.png` shows it covering the MTU
 * reading). A control that obscures the data it is meant to refresh is the
 * defect; the phone is where it bites, because the band and the fold are close
 * together.
 *
 * The assertion is geometric — the two boxes must not intersect — because that
 * is the failure an operator sees and the one no amount of copy can work
 * around. The FAB's accessible name is asserted alongside it: the control was
 * icon-only, so a screen-reader user and a tooltip-less touch user both had to
 * guess what the circle did.
 */

import { expect, test } from '@playwright/test';

import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';

const PHONE = { width: 390, height: 844 };

test.use({ storageState: AUTH_STORAGE_STATE });

test.describe('the run control at phone width', () => {
  test.beforeEach(async ({ page }) => {
    await disableAnimations(page);
    await page.setViewportSize(PHONE);
    await page.goto('/link', { waitUntil: 'domcontentloaded' });
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
  });

  test('does not overlap the status band figures', async ({ page }) => {
    // The rollup band is the page's own `aria-live` section; the figures are
    // the Speed / Duplex / MTU readings inside it.
    const band = page.locator('section[aria-live="polite"][data-state]');
    await expect(band).toBeVisible();

    const run = page.getByRole('button', { name: /run all tests/i });
    await expect(run).toBeVisible();

    const bandBox = await band.boundingBox();
    const runBox = await run.boundingBox();
    expect(bandBox, 'status band should have a box').not.toBeNull();
    expect(runBox, 'run control should have a box').not.toBeNull();
    if (!bandBox || !runBox) return;

    const overlaps =
      runBox.x < bandBox.x + bandBox.width &&
      runBox.x + runBox.width > bandBox.x &&
      runBox.y < bandBox.y + bandBox.height &&
      runBox.y + runBox.height > bandBox.y;

    expect(
      overlaps,
      `run control [${Math.round(runBox.x)},${Math.round(runBox.y)} ${Math.round(runBox.width)}x${Math.round(runBox.height)}] ` +
        `intersects the status band [${Math.round(bandBox.x)},${Math.round(bandBox.y)} ${Math.round(bandBox.width)}x${Math.round(bandBox.height)}]`,
    ).toBe(false);
  });

  test('carries a visible label, not just an icon', async ({ page }) => {
    const run = page.getByRole('button', { name: /run all tests/i });
    await expect(run).toBeVisible();

    // An accessible name alone was already true of the icon-only FAB. What the
    // row asks for is a label the sighted touch user can read without hovering,
    // which a phone cannot do at all.
    await expect(run).toContainText(/run all tests/i);
  });
});
