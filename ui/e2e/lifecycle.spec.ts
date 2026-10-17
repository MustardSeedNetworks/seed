import { expect, type Page, test } from '@playwright/test';
import { sidebarSettingsButton, skipSetupWizard } from './helpers/auth';

/**
 * Hardware lifecycle (UI-SEED-47), against the E2E daemon.
 *
 * An operator imports a Cisco EoX API response from the page and the daemon
 * reports the products it now holds. The browser projects share one daemon,
 * so both import the same file and the result is the same whichever runs
 * last; nothing else in the suite reads the EoX table.
 *
 * The admin's feed answer is saved as off, the default, and turning it on
 * without a Cisco API client is refused with the server's reason: neither
 * leaves the daemon able to call out.
 */

const EOX_RESPONSE = {
  PaginationResponseRecord: { PageIndex: 1, LastIndex: 1, TotalRecords: 2, PageRecords: 2 },
  EOXRecord: [
    {
      EOLProductID: 'WS-C3750X-48P-S',
      ProductIDDescription: 'Catalyst 3750X 48 Port PoE IP Base',
      EndOfSaleDate: { value: '2016-10-30', dateFormat: 'YYYY-MM-DD' },
      LastDateOfSupport: { value: '2021-10-31', dateFormat: 'YYYY-MM-DD' },
      EOXMigrationDetails: { MigrationProductId: 'WS-C3850-48P-S' },
    },
    {
      EOLProductID: 'C9300-48P',
      EndOfSaleDate: { value: '2026-09-01', dateFormat: 'YYYY-MM-DD' },
      LastDateOfSupport: { value: '2031-09-30', dateFormat: 'YYYY-MM-DD' },
      EOXMigrationDetails: { MigrationProductId: '' },
    },
  ],
};

async function openFeed(page: Page): Promise<void> {
  await page.goto('/');
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
  await sidebarSettingsButton(page).click();
  await expect(page.getByTestId('settings-drawer')).toBeVisible({ timeout: 5000 });
  const section = page.getByTestId('eox-feed-section');
  await section.getByRole('button', { expanded: false }).first().click();
  await expect(page.getByTestId('eox-feed-save')).toBeEnabled();
}

async function saveFeed(page: Page): Promise<number> {
  const put = page.waitForResponse(
    (r) => r.request().method() === 'PUT' && r.url().endsWith('/api/v1/lifecycle/feed'),
  );
  await page.getByTestId('eox-feed-save').click();
  return (await put).status();
}

test.describe('Hardware lifecycle', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('an operator imports an EoX file and the daemon holds its products', async ({ page }) => {
    await page.goto('/lifecycle');
    await expect(page.getByTestId('lifecycle-import')).toBeEnabled({ timeout: 10000 });

    const put = page.waitForResponse(
      (r) => r.request().method() === 'PUT' && r.url().endsWith('/api/v1/lifecycle/eox'),
    );
    await page.getByTestId('lifecycle-import-file').setInputFiles({
      name: 'eox.json',
      mimeType: 'application/json',
      buffer: Buffer.from(JSON.stringify(EOX_RESPONSE)),
    });
    expect((await put).status(), 'PUT /api/v1/lifecycle/eox').toBe(200);
    await expect(page.getByTestId('lifecycle-import-result')).toHaveText(
      'Imported EoX data for 2 products.',
    );

    await page.reload();
    await expect(page.getByTestId('lifecycle-counts')).toContainText('EoX data for 2 products');
  });

  test('a file that is not an EoX response is refused and named', async ({ page }) => {
    await page.goto('/lifecycle');
    await expect(page.getByTestId('lifecycle-import')).toBeEnabled({ timeout: 10000 });

    const put = page.waitForResponse(
      (r) => r.request().method() === 'PUT' && r.url().endsWith('/api/v1/lifecycle/eox'),
    );
    await page.getByTestId('lifecycle-import-file').setInputFiles({
      name: 'not-eox.json',
      mimeType: 'application/json',
      buffer: Buffer.from(JSON.stringify({ products: [] })),
    });
    expect((await put).status(), 'PUT /api/v1/lifecycle/eox').toBe(400);
    await expect(page.getByTestId('lifecycle-import-result')).toHaveText(
      'That file is not a Cisco EoX API response.',
    );
  });

  test('an admin keeps the feed off, and on without a client is refused', async ({ page }) => {
    // Three loads of the Settings drawer; webkit on CI needs ~7 s each.
    test.slow();
    await openFeed(page);
    await page.getByTestId('eox-feed-off').check();
    expect(await saveFeed(page), 'PUT /api/v1/lifecycle/feed off').toBe(200);

    await openFeed(page);
    await expect(page.getByTestId('eox-feed-off')).toBeChecked();
    await expect(page.getByTestId('eox-feed-unanswered')).toHaveCount(0);

    await page.getByTestId('eox-feed-on').check();
    expect(await saveFeed(page), 'PUT /api/v1/lifecycle/feed on').toBe(400);
    await expect(page.getByTestId('eox-feed-error')).toContainText(
      'the Cisco EoX API needs a client ID and secret',
    );

    await openFeed(page);
    await expect(page.getByTestId('eox-feed-off')).toBeChecked();
  });
});
