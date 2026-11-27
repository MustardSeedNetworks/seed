import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * The MIB browser on a polling target (UI-SEED-49).
 *
 * The first test walks a name the MIB database does not know and reads the
 * daemon's refusal; the second walks a truncated ifDescr subtree to show the
 * rows and the limit note.
 */

const targetId = 'e2e-mib-target';

/**
 * The page's target list and the walk route are both answered here. A free
 * licence allows one polling target, and other specs create real ones in
 * parallel workers, so a target made through the API for this spec answers 402
 * whenever it loses that race. The walk body is what this spec is about, and
 * it is a wire shape the daemon's own handler tests pin.
 */
async function routeTarget(page: Page): Promise<void> {
  const now = new Date().toISOString();
  await page.route('**/api/v1/polling-targets', (route) =>
    route.request().method() === 'GET'
      ? route.fulfill({
          json: {
            count: 1,
            targets: [
              {
                id: targetId,
                clientId: 'e2e',
                name: 'e2e-mib-target',
                ipAddress: '192.0.2.20',
                snmpVersion: 'v2c',
                credentialsId: '',
                pollIntervalSeconds: 60,
                enabled: false,
                collectorChain: [],
                lastStatus: '',
                lastError: '',
                createdAt: now,
                updatedAt: now,
              },
            ],
          },
        })
      : route.continue(),
  );
}

test.describe('MIB browser', () => {
  test.beforeEach(async ({ page }) => {
    await routeTarget(page);
    await skipSetupWizard(page);
    await page.goto('/polling-targets');
    await page.getByTestId(`target-row-${targetId}`).click();
  });

  test('an unknown object name reads as the daemon words it', async ({ page }) => {
    await page.route(`**/api/v1/polling-targets/${targetId}/walk`, (route) =>
      route.fulfill({
        status: 400,
        json: {
          error: "'oid' must be a numeric OID or a known MIB object name",
          code: 'VALIDATION_ERROR',
        },
      }),
    );
    const browser = page.getByTestId('mib-browser');
    await browser.getByTestId('mib-oid').fill('notAnObjectAnyMibDefines');
    await browser.getByTestId('mib-walk').click();
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
