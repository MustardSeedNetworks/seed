import { type APIRequestContext, expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Device groups (UI-SEED-25), against the E2E daemon: an operator builds a
 * group from a polling target, and after a reload the group, its member and
 * the member's tick in the edit form are what the daemon stored. Then the
 * group is deleted from the page.
 *
 * The target is made through the API before the page loads: minting a CSRF
 * token replaces the session's token, so doing it after the page has cached
 * one would refuse the page's own writes.
 */

async function csrf(request: APIRequestContext): Promise<Record<string, string>> {
  const response = await request.get('/api/v1/auth/csrf');
  expect(response.status(), 'GET /api/v1/auth/csrf').toBe(200);
  const { token } = (await response.json()) as { token: string };
  return { 'X-CSRF-Token': token };
}

test('an operator groups a polling target and the membership round-trips', async ({
  page,
  request,
}) => {
  const suffix = Date.now().toString(36);
  const targetName = `e2e-group-member-${suffix}`;
  const groupName = `E2E group ${suffix}`;
  // Disabled, so the poller never tries it and it needs no credential.
  const created = await request.post('/api/v1/polling-targets', {
    headers: await csrf(request),
    data: { name: targetName, ipAddress: '192.0.2.10', snmpVersion: 'v2c', enabled: false },
  });
  expect(created.status(), 'POST /api/v1/polling-targets').toBe(200);
  const { id: targetId } = (await created.json()) as { id: string };

  try {
    await skipSetupWizard(page);
    await page.goto('/device-groups');
    const add = page.getByTestId('group-add');
    await expect(add).toBeEnabled({ timeout: 10000 });
    await add.click();

    const dialog = page.getByRole('dialog', { name: 'Add device group' });
    await dialog.getByTestId('group-name').fill(groupName);
    await dialog.getByTestId(`group-member-${targetId}`).check();
    const saved = page.waitForResponse(
      (r) => r.request().method() === 'POST' && r.url().endsWith('/api/v1/device-groups'),
    );
    await dialog.getByTestId('group-save').click();
    expect((await saved).status(), 'POST /api/v1/device-groups').toBe(201);
    await expect(dialog).toBeHidden();

    await page.reload();
    const row = page.getByRole('button', { name: new RegExp(groupName) });
    await expect(row).toContainText('1 member');
    await row.click();
    await expect(page.getByTestId(`group-summary-member-${targetId}`)).toContainText(targetName);
    // A disabled target is not polled, so it is unknown, never healthy.
    await expect(page.getByTestId(`group-summary-member-${targetId}`)).toHaveAttribute(
      'data-state',
      'unknown',
    );

    await page.getByTestId('group-edit').click();
    const edit = page.getByRole('dialog', { name: 'Edit device group' });
    await expect(edit.getByTestId(`group-member-${targetId}`)).toBeChecked();
    await edit.getByRole('button', { name: 'Cancel' }).click();

    page.once('dialog', (confirm) => void confirm.accept());
    await page.getByTestId('group-delete').click();
    await expect(row).toBeHidden();
    await page.reload();
    await expect(page.getByRole('button', { name: new RegExp(groupName) })).toBeHidden();
  } finally {
    await request.delete(`/api/v1/polling-targets/${targetId}`, {
      headers: await csrf(request),
    });
  }
});
