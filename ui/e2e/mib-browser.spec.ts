import { type APIRequestContext, expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * The MIB browser on a polling target (UI-SEED-49), against the E2E daemon.
 *
 * The daemon reaches no SNMP agent, so the first test walks a name the MIB
 * database does not know and reads the daemon's own refusal. The second routes
 * the walk to a truncated ifDescr subtree to show the rows and the limit note.
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

test.describe('MIB browser', () => {
  let targetId = '';
  let targetName = '';

  test.beforeEach(async ({ page, request }) => {
    targetName = `e2e-mib-${Date.now().toString(36)}`;
    // Disabled, so the poller never tries it and it needs no credential.
    const created = await request.post('/api/v1/polling-targets', {
      headers: await csrf(request),
      data: { name: targetName, ipAddress: '192.0.2.20', snmpVersion: 'v2c', enabled: false },
    });
    expect(created.status(), 'POST /api/v1/polling-targets').toBe(200);
    ({ id: targetId } = (await created.json()) as { id: string });

    await skipSetupWizard(page);
    await page.goto('/polling-targets');
    await page.getByTestId(`target-row-${targetId}`).click();
  });

  test.afterEach(async ({ request }) => {
    await request.delete(`/api/v1/polling-targets/${targetId}`, {
      headers: await csrf(request),
    });
  });

  test('an unknown object name reads as the daemon words it', async ({ page }) => {
    const browser = page.getByTestId('mib-browser');
    await browser.getByTestId('mib-oid').fill('notAnObjectAnyMibDefines');
    const walked = page.waitForResponse((r) => r.url().endsWith(`/${targetId}/walk`));
    await browser.getByTestId('mib-walk').click();
    expect((await walked).status(), 'POST .../walk').toBe(400);
    await expect(browser.getByTestId('mib-error')).toHaveText(
      "The walk failed: API error: 400: 'oid' must be a numeric OID or a known MIB object name",
    );
  });

  test('a walk lists its rows and says when it stopped at the limit', async ({ page }) => {
    const sent: unknown[] = [];
    await page.route(`**/api/v1/polling-targets/${targetId}/walk`, (route) => {
      sent.push(route.request().postDataJSON());
      return route.fulfill({
        json: {
          targetId,
          oid: '1.3.6.1.2.1.2.2.1.2',
          truncated: true,
          rows: [
            { oid: '1.3.6.1.2.1.2.2.1.2.1', name: 'ifDescr', instance: '1', value: 'Gi0/1' },
            { oid: '1.3.6.1.2.1.2.2.1.2.2', name: 'ifDescr', instance: '2', value: 'Gi0/2' },
          ],
        },
      });
    });

    const browser = page.getByTestId('mib-browser');
    await browser.getByTestId('mib-oid').fill('ifDescr');
    await browser.getByTestId('mib-oid').press('Enter');

    await expect(browser.getByTestId('mib-summary')).toHaveText(
      '2 objects under 1.3.6.1.2.1.2.2.1.2',
    );
    await expect(browser.getByTestId('mib-truncated')).toBeVisible();
    await expect(browser.getByTestId('mib-rows').getByRole('listitem')).toHaveText([
      'ifDescr.1Gi0/1',
      'ifDescr.2Gi0/2',
    ]);
    expect(sent).toEqual([{ oid: 'ifDescr' }]);
  });
});
