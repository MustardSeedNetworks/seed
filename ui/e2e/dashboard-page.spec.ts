import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Dashboard (UI-SEED-22): the signed-in user's own page of cards. A layout
 * is added to, reordered and pruned in a draft, stored on Save, and read back
 * from the account rather than the browser. Runs against the real daemon;
 * per-user isolation is pinned by the API tests.
 */

const DEFAULT = ['link', 'gateway', 'dns', 'network'];

async function shown(page: Page): Promise<string[]> {
  return page
    .getByTestId(/^dashboard-widget-/)
    .evaluateAll((els) =>
      els.map((el) => el.getAttribute('data-testid')?.replace('dashboard-widget-', '') ?? ''),
    );
}

async function saveLayout(page: Page): Promise<void> {
  const saved = page.waitForResponse(
    (r) => r.url().endsWith('/api/v1/users/me/dashboard') && r.request().method() === 'PUT',
  );
  await page.getByTestId('dashboard-save').click();
  expect((await saved).status()).toBe(200);
  await expect(page.getByTestId('dashboard-editor')).toBeHidden();
}

test.describe.configure({ mode: 'serial' });

test.beforeEach(async ({ page }) => {
  await skipSetupWizard(page);
  await page.goto('/dashboard');
  // Start from the default whatever an earlier run saved for this account.
  await page.getByTestId('dashboard-customize').click();
  await page.getByTestId('dashboard-reset').click();
  await saveLayout(page);
});

test('adds, reorders and removes cards, and the layout outlives the browser', async ({ page }) => {
  expect(await shown(page)).toEqual(DEFAULT);

  await page.getByTestId('dashboard-customize').click();
  await page.getByTestId('dashboard-add-choice').selectOption('publicIp');
  await page.getByTestId('dashboard-add').click();
  await page.getByTestId('dashboard-up-publicIp').click();
  await page.getByTestId('dashboard-remove-dns').click();
  await page.getByTestId('dashboard-down-link').click();
  const draft = ['gateway', 'link', 'publicIp', 'network'];
  await expect.poll(() => shown(page)).toEqual(draft);
  await saveLayout(page);

  // Nothing of the layout lives in the browser: with local storage cleared,
  // a fresh load still reads it from the account.
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await expect(page.getByTestId('dashboard-widget-gateway')).toBeAttached();
  expect(await shown(page)).toEqual(draft);
  await expect(page.getByTestId('dashboard-widget-publicIp').getByTestId('card')).toBeVisible();
});

test('cancel leaves the saved layout untouched', async ({ page }) => {
  await page.getByTestId('dashboard-customize').click();
  await page.getByTestId('dashboard-remove-link').click();
  await page.getByTestId('dashboard-cancel').click();
  expect(await shown(page)).toEqual(DEFAULT);

  await page.reload();
  await expect(page.getByTestId('dashboard-widget-link')).toBeAttached();
  expect(await shown(page)).toEqual(DEFAULT);
});
