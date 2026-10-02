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

import { type APIRequestContext, expect, test } from '@playwright/test';

import { AUTH_STORAGE_STATE, disableAnimations, sidebarSettingsButton } from './helpers/auth';

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

async function putIdentity(
  request: APIRequestContext,
  identity: { name: string; location: string },
): Promise<{ status: number; body: unknown }> {
  const csrf = await request.get('/api/v1/auth/csrf');
  expect(csrf.status(), 'GET /api/v1/auth/csrf').toBe(200);
  const { token } = (await csrf.json()) as { token: string };
  const response = await request.put('/api/v1/settings', {
    headers: { 'X-CSRF-Token': token },
    data: { identity },
  });
  return { status: response.status(), body: await response.json() };
}

test.describe('a route names itself the same way everywhere', () => {
  // The device-name case below renames the daemon every page reads, which
  // changes the title the per-route cases assert. Serial keeps it from
  // running between them; it restores the empty identity when it is done.
  test.describe.configure({ mode: 'serial' });

  for (const { path, label } of PAGES) {
    test(`${path} titles the tab and the breadcrumb with its own label`, async ({ page }) => {
      await disableAnimations(page);
      await page.goto(path, { waitUntil: 'domcontentloaded' });
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });

      // The tab. Nothing set this per route before; it read the bare product
      // name on all eleven, so several open tabs were indistinguishable. The
      // suffix is asserted too: it is the one place the product name reaches
      // the browser chrome, and UI-SEED-9 dropped the article from it.
      await expect(page).toHaveTitle(`${label} · Seed`);

      // The breadcrumb's last crumb is the current page, and it has to be the
      // same string as the label — not a de-slugged URL segment.
      const crumb = page.getByRole('navigation', { name: 'Breadcrumb' }).getByText(label, {
        exact: true,
      });
      await expect(crumb).toBeVisible();
    });
  }

  test('a named device carries its name in the rail and every tab title', async ({ page }) => {
    await disableAnimations(page);
    await page.goto('/link', { waitUntil: 'domcontentloaded' });
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
    // Unnamed, the tab is page then product and the rail has no name line.
    await expect(page).toHaveTitle('Link · Seed');
    await expect(page.getByTestId('rail-device-name')).toHaveCount(0);

    try {
      await sidebarSettingsButton(page).click();
      await page.getByTestId('device-identity-settings-section').click();
      await page.getByTestId('device-identity-name').fill('seed-idf-3b');
      await page.getByTestId('device-identity-location').fill('Main Office, IDF 3B, Port 21');
      await page.getByTestId('device-identity-save').click();

      // The rail and the tab follow the save, with no reload.
      await expect(page.getByTestId('rail-device-name')).toHaveText('seed-idf-3b');
      await expect(page).toHaveTitle('Link · seed-idf-3b · Seed');

      // It is the daemon's, not this browser's: a fresh load on another
      // route reads it back from the server.
      await page.goto('/network', { waitUntil: 'domcontentloaded' });
      await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
      await expect(page).toHaveTitle('Network · seed-idf-3b · Seed');
      await expect(page.getByTestId('rail-device-name')).toHaveText('seed-idf-3b');
      const settings = await page.request.get('/api/v1/settings');
      expect(await settings.json()).toMatchObject({
        identity: { name: 'seed-idf-3b', location: 'Main Office, IDF 3B, Port 21' },
      });

      // A name the tab title could not show as written is refused with the
      // reason, and the stored one stands.
      const refused = await putIdentity(page.request, { name: 'rack\n12', location: '' });
      expect(refused.status).toBe(400);
      expect(JSON.stringify(refused.body)).toContain('identity.name must be one line of text');
      const after = await page.request.get('/api/v1/settings');
      expect(await after.json()).toMatchObject({ identity: { name: 'seed-idf-3b' } });
    } finally {
      // Puts the daemon's identity back, so no later spec sees this rename.
      const cleared = await putIdentity(page.request, { name: '', location: '' });
      expect(cleared.status, 'PUT /api/v1/settings').toBe(200);
    }
  });

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
