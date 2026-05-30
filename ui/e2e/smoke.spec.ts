import { expect, test } from '@playwright/test';
import { AUTH_STORAGE_STATE, sidebarHelpButton, sidebarSettingsButton } from './helpers/auth';

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

  // No top-level theme-toggle smoke test: seed's data-testid="theme-toggle"
  // lives on AppearanceSettings.tsx — only mounted when the settings
  // drawer is open. Theme behaviour is covered by theme-and-help.spec.ts
  // at the @smoke tier (reached through the drawer). Putting a deep-link
  // assertion at the top-level smoke tier was a mis-port of stem's smoke
  // shape (stem has header-theme-toggle at the chrome level).

  test('settings drawer opens from sidebar', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await sidebarSettingsButton(page).click();
    await expect(page.getByTestId('settings-drawer')).toBeVisible();
    await page.getByTestId('settings-drawer-close').click();
    await expect(page.getByTestId('settings-drawer')).toBeHidden();
  });

  test('help drawer opens from sidebar', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await sidebarHelpButton(page).click();
    await expect(page.getByTestId('help-drawer')).toBeVisible();
    await expect(page.getByTestId('help-drawer-content')).toBeVisible();
    await page.getByTestId('help-drawer-close').click();
    await expect(page.getByTestId('help-drawer')).toBeHidden();
  });

  test('profile dropdown reveals logout control', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await page.getByTestId('header-profile').click();
    await expect(page.getByTestId('header-logout')).toBeVisible({ timeout: 5000 });
  });
});
