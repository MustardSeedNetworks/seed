import { expect, test } from '@playwright/test';

/**
 * Dashboard E2E Tests (@smoke)
 *
 * Asserts the load-bearing dashboard chrome that's visible regardless
 * of backend data availability:
 *   - active route H1
 *   - sidebar nav buttons for every module group
 *   - settings + help drawer triggers
 *
 * Removed the per-card assertions (Link Status / Gateway / DNS) — those
 * depend on backend permissions and discovery state that aren't
 * guaranteed in an unprivileged CI runner (macOS dev box can't open
 * ICMP sockets without sudo, runner has no real network). Card-level
 * coverage lives in tier-2 integration specs.
 *
 * Suite-wide storageState (e2e/global-setup.ts) handles auth — no
 * mockAuthenticated needed.
 */

test.describe('@smoke Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // H1 "Link" is the active route; level: 1 + exact-match disambiguates
    // from the H3 "Link Status" card chrome that would trip strict mode.
    await expect(page.getByRole('heading', { name: /^link$/i, level: 1 })).toBeVisible({
      timeout: 10000,
    });
  });

  test('renders all sidebar module nav buttons', async ({ page }) => {
    // Each Seed module group surfaces one or more buttons in the
    // sidebar. Asserting their presence catches sidebar regressions
    // (renames, accidental removal, layout break) without coupling to
    // backend data.
    for (const name of [
      'Link',
      'Network',
      'Path Analysis',
      'Wi-Fi',
      'Security',
      'Performance',
      'Reports',
      'Logs',
    ]) {
      await expect(page.getByRole('button', { name, exact: true })).toBeVisible();
    }
  });

  test('opens settings drawer', async ({ page }) => {
    await page.getByRole('button', { name: 'Open settings' }).first().click();
    // Settings drawer surfaces several section headers; thresholds is
    // the most stable across module config refactors.
    await expect(page.getByText(/thresholds|appearance|discovery/i).first()).toBeVisible({
      timeout: 5000,
    });
  });

  test('opens help drawer', async ({ page }) => {
    await page.getByRole('button', { name: 'Open help' }).first().click();
    await expect(page.getByRole('dialog').or(page.locator('[role="dialog"]'))).toBeVisible({
      timeout: 5000,
    });
  });
});
