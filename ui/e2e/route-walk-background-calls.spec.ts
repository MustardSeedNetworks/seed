/**
 * route-walk-background-calls.spec.ts — walking the routes spends nothing an
 * operator needs (D-SEED-20, seed#2691).
 *
 * Sixteen routes share one 5-per-minute rate-limit bucket per client. Two page
 * mounts spent it: the app posted `/security/devices/scan` after sign-in, and
 * the Health Check card ran `/telemetry/probes/run` on every Performance visit.
 * After a short walk, the operator's own capture start answered 429. The cable
 * poll also ran where the platform has no cable test, logging a 501 each time.
 *
 * One walk through the sidebar, so it is client-side navigation over a single
 * app load, the way an operator moves between pages.
 */

import { expect, test } from '@playwright/test';

import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';

/** Every `pageRegistry` route, in sidebar order. */
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

/** Rate-limited routes a page mount used to call (server_routes.go `rateLimited`). */
const MOUNT_SPENDERS = /\/api\/v1\/(security\/devices\/scan|telemetry\/probes\/run)(\?|$)/;

test.use({ storageState: AUTH_STORAGE_STATE, viewport: { width: 1440, height: 900 } });

test('a walk through every route calls no rate-limited route and logs no 501 or 429', async ({
  page,
}) => {
  // Eleven navigations, plus the sign-in scan's old two-second delay, do not
  // fit the suite's 30 s per-test budget in webkit.
  test.setTimeout(120_000);
  await disableAnimations(page);

  const spent: string[] = [];
  page.on('request', (request) => {
    if (MOUNT_SPENDERS.test(request.url())) {
      spent.push(`${request.method()} ${new URL(request.url()).pathname}`);
    }
  });
  const refused: string[] = [];
  page.on('response', (response) => {
    if (response.status() === 501 || response.status() === 429) {
      refused.push(`${response.status()} ${new URL(response.url()).pathname}`);
    }
  });
  const consoleErrors: string[] = [];
  page.on('console', (message) => {
    if (message.type() === 'error' && /\b(501|429)\b/.test(message.text())) {
      consoleErrors.push(message.text());
    }
  });

  await page.goto('/link', { waitUntil: 'domcontentloaded' });
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 20000 });
  for (const route of ROUTES) {
    await page.getByTestId(`sidebar-nav-${route.slice(1)}`).click();
    await expect(page).toHaveURL(new RegExp(`${route}$`));
    await expect(page.getByTestId('page-header-title')).toBeVisible();
  }
  // Back to Performance last: the Health Check card has mounted again and
  // says it has not run, which is the state its old mount run replaced.
  await page.getByTestId('sidebar-nav-performance').click();
  await expect(page.getByTestId('health-check-run')).toBeEnabled();

  expect(spent, 'rate-limited calls made by page mounts').toEqual([]);
  expect(refused, '501/429 responses during the walk').toEqual([]);
  expect(consoleErrors, '501/429 console errors during the walk').toEqual([]);
});
