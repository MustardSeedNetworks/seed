/**
 * route-identity.spec.ts — a route names itself the same way everywhere.
 *
 * Navigation labels used to be defined in three places and disagree (#2645):
 * the locale file gave `/network` an eyebrow of "Diagnostics" while the rail
 * filed it under Live Telemetry; `Breadcrumbs.tsx` hard-coded a label map that
 * knew nine of the eleven routes, so `/polling-targets` fell through to a
 * de-slugged segment and read "Polling Targets" over an H1 of "Polling
 * targets"; and `document.title` was never set per route at all, so every tab,
 * bookmark and history entry read the bare product name.
 *
 * All three now derive from `pageRegistry`. The unit tests
 * (`navGroups.test.ts`, `pageRegistry.test.tsx`) pin the data; this asserts
 * what the operator actually sees in a browser, which is the part a data-level
 * test cannot reach — `document.title` in particular is set by an effect and
 * has no rendered representation.
 */

import { expect, test } from '@playwright/test';

import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';

/**
 * Every page in `pageRegistry`, with the label the registry resolves for it.
 * A subset would only prove the subset — and the three routes the old label
 * map was missing are exactly the ones worth asserting.
 */
const PAGES = [
  { path: '/link', label: 'Link' },
  { path: '/network', label: 'Network' },
  { path: '/path', label: 'Path Analysis' },
  { path: '/wifi', label: 'Wi-Fi' },
  { path: '/security', label: 'Security' },
  { path: '/performance', label: 'Performance' },
  { path: '/reports', label: 'Reports' },
  { path: '/logs', label: 'Logs' },
  { path: '/polling-targets', label: 'Polling targets' },
  { path: '/topology', label: 'Topology' },
  { path: '/alerts', label: 'Alerts' },
];

test.use({ storageState: AUTH_STORAGE_STATE });

test.describe('a route names itself the same way everywhere', () => {
  for (const { path, label } of PAGES) {
    test(`${path} titles the tab and the breadcrumb with its own label`, async ({ page }) => {
      await disableAnimations(page);
      await page.goto(path, { waitUntil: 'domcontentloaded' });
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });

      // The tab. Nothing set this per route before; it read "The Seed" on all
      // eleven, so several open tabs were indistinguishable.
      await expect(page).toHaveTitle(new RegExp(`^${label} · `));

      // The breadcrumb's last crumb is the current page, and it has to be the
      // same string as the label — not a de-slugged URL segment.
      const crumb = page.getByRole('navigation', { name: 'Breadcrumb' }).getByText(label, {
        exact: true,
      });
      await expect(crumb).toBeVisible();
    });
  }

  test('the eyebrow over the title is the rail group the route sits in', async ({ page }) => {
    await disableAnimations(page);

    // /network is the case #2645 names: its eyebrow read "Diagnostics" while
    // the rail filed it under Live Telemetry.
    await page.goto('/network', { waitUntil: 'domcontentloaded' });
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('page-header-eyebrow')).toHaveText('Live Telemetry');

    await page.goto('/wifi', { waitUntil: 'domcontentloaded' });
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('page-header-eyebrow')).toHaveText('Diagnostics');
  });
});
