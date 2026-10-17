import { type APIRequestContext, expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * IP addresses (UI-SEED-46), against the E2E daemon: an operator adds a DHCP
 * scope from the page, and after a reload the scope and its counts are what
 * the daemon stored. Then the scope is deleted from the page.
 *
 * Scopes may not overlap, and the browser projects share one daemon, so each
 * concurrent worker takes its own slice of TEST-NET-2.
 */

async function csrf(request: APIRequestContext): Promise<Record<string, string>> {
  const response = await request.get('/api/v1/auth/csrf');
  expect(response.status(), 'GET /api/v1/auth/csrf').toBe(200);
  const { token } = (await response.json()) as { token: string };
  return { 'X-CSRF-Token': token };
}

test('an operator adds a DHCP scope, sees what the daemon counts and deletes it', async ({
  page,
  request,
}, testInfo) => {
  const base = 10 + testInfo.parallelIndex * 20;
  const start = `198.51.100.${base}`;
  const end = `198.51.100.${base + 10}`;
  const name = `E2E scope ${Date.now().toString(36)}`;

  try {
    await skipSetupWizard(page);
    await page.goto('/ipam');
    const add = page.getByTestId('scope-add');
    await expect(add).toBeEnabled({ timeout: 10000 });
    await add.click();

    const dialog = page.getByRole('dialog', { name: 'Add DHCP scope' });
    await dialog.getByTestId('scope-name').fill(name);
    await dialog.getByTestId('scope-start').fill(start);
    await dialog.getByTestId('scope-end').fill(end);
    const saved = page.waitForResponse(
      (r) => r.request().method() === 'POST' && r.url().endsWith('/api/v1/ipam/scopes'),
    );
    await dialog.getByTestId('scope-save').click();
    expect((await saved).status(), 'POST /api/v1/ipam/scopes').toBe(200);
    await expect(dialog).toBeHidden();

    await page.reload();
    const row = page.getByTestId(`ipam-scope-${start}`);
    await expect(row).toContainText(`${start} – ${end}`);
    await expect(row).toContainText(name);
    await row.click();
    await expect(page.getByTestId('ipam-size')).toHaveText(/^Addresses\s*11$/);
    // Nothing in TEST-NET-2 answers, so no address of the scope is in use.
    await expect(page.getByTestId('ipam-used')).toHaveText(/^In use\s*0$/);
    await expect(page.getByTestId('ipam-in-use-empty')).toBeVisible();

    page.once('dialog', (confirm) => void confirm.accept());
    await page.getByTestId('scope-delete').click();
    await expect(row).toBeHidden();
    await page.reload();
    await expect(page.getByTestId('ipam-counts')).toBeVisible();
    await expect(page.getByTestId(`ipam-scope-${start}`)).toBeHidden();
  } finally {
    // A no-op 404 when the page already deleted it.
    await request.delete(`/api/v1/ipam/scopes?start=${start}`, { headers: await csrf(request) });
  }
});
