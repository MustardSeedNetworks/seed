import { expect, test } from '@playwright/test';
import { disableAnimations, TEST_CREDENTIALS } from './helpers/auth';

/**
 * First-run EoX feed question (UI-SEED-47, seed#3460), against the E2E daemon.
 *
 * Global setup has already completed first run on the shared daemon, so this
 * stands in for a fresh install with three routes: setup status says setup is
 * needed, setup completion succeeds, and the feed reads as unanswered. The
 * sign-in that follows and the saved answer are the daemon's own.
 *
 * The answer is saved as off, the default; lifecycle.spec.ts saves the same.
 */

test.use({ storageState: { cookies: [], origins: [] } });

test('setup ends by asking the EoX feed question, answered off by default', async ({ page }) => {
  await disableAnimations(page);
  await page.route(/\/api\/v1\/setup\/status$/, (route) =>
    route.fulfill({
      json: { needsSetup: true, username: TEST_CREDENTIALS.username, setupTokenFile: '/tmp/t' },
    }),
  );
  await page.route(/\/api\/v1\/setup\/complete$/, (route) =>
    route.fulfill({ json: { success: true } }),
  );
  await page.route(/\/api\/v1\/lifecycle\/feed$/, (route) =>
    route.request().method() === 'GET'
      ? route.fulfill({
          json: {
            choice: 'unanswered',
            endpoint: '',
            tokenUrl: '',
            clientId: '',
            hasClientSecret: false,
            fetched: 0,
          },
        })
      : route.continue(),
  );

  await page.goto('/');
  await page.getByLabel('Setup token').fill('fresh-install-token');
  await page.getByLabel('Password', { exact: true }).fill(TEST_CREDENTIALS.password);
  await page.getByLabel('Confirm Password').fill(TEST_CREDENTIALS.password);
  await page.getByRole('button', { name: 'Complete Setup' }).click();
  await page.getByRole('button', { name: 'Skip for now' }).click();

  await expect(page.getByTestId('setup-eox-step')).toBeVisible();
  await expect(page.getByTestId('eox-feed-off')).toBeChecked();
  await expect(page.getByTestId('setup-eox-save')).toBeEnabled();

  const put = page.waitForResponse(
    (r) => r.request().method() === 'PUT' && r.url().endsWith('/api/v1/lifecycle/feed'),
  );
  await page.getByTestId('setup-eox-save').click();
  const response = await put;
  expect(response.status(), 'PUT /api/v1/lifecycle/feed').toBe(200);
  expect(((await response.json()) as { choice: string }).choice).toBe('off');

  await expect(page.getByTestId('setup-eox-step')).toBeHidden();
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
});
