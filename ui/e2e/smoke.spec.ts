import { expect, test } from '@playwright/test';
import { AUTH_STORAGE_STATE } from './helpers/auth';

const VERSION_KEYS = ['version', 'commit', 'buildTime', 'uiBuildHash'] as const;

test.describe('smoke @ unauthenticated', { tag: '@smoke' }, () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test('GET /__version returns canonical build metadata', async ({ request }) => {
    const res = await request.get('/__version');
    expect(res.status()).toBe(200);
    const body = await res.json();
    for (const k of VERSION_KEYS) {
      expect(body[k], `missing ${k} in /__version`).toBeTruthy();
      expect(typeof body[k]).toBe('string');
    }
  });

  test('login surface renders for unauthenticated visitors', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('login-title')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('smoke @ authenticated', { tag: '@smoke' }, () => {
  test.use({ storageState: AUTH_STORAGE_STATE });

  test('dashboard renders with page header and at least one card', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('card').first()).toBeVisible();
  });

  // Deliberately no drawer, theme or logout tests here. Each was a second
  // copy of a behaviour already owned elsewhere, and two of them ran in this
  // very job:
  //
  //   settings drawer open/close -> settings.spec.ts opens it in beforeEach
  //                                 and closes it in its own test
  //   help drawer open/close     -> theme-and-help.spec.ts, whose describe is
  //                                 itself tagged @smoke, so its 14 theme and
  //                                 help tests already run in this job — and
  //                                 assert role, aria-modal, the TOC and ESC,
  //                                 which the copy here did not
  //   profile -> logout          -> auth-complete.spec.ts drives the same
  //                                 header-profile click as its logout setup
  //
  // What stays is what nothing else covers: the build-metadata contract, the
  // unauthenticated login surface, and that an authenticated dashboard paints.
  // A smoke tier earns its place by being the fast boot check, not by being a
  // sample of the full suite.
});
