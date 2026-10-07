import { expect, type Page, test } from '@playwright/test';
import { sidebarSettingsButton, skipSetupWizard } from './helpers/auth';

/**
 * Alert escalation ladders (#3186), against the E2E daemon.
 *
 * The ladder editor saves alerts.escalations through PUT /api/v1/settings,
 * which the running daemon applies without a restart, so a reload must serve
 * back what was saved. A ladder the server cannot run is refused with its
 * reason, shown as given. An escalated alert needs a ladder to have run for
 * at least a minute, so the inbox half serves a fixture alert instead.
 *
 * The suite's daemon is shared, so the test ends with no ladders stored.
 */

async function openEscalation(page: Page): Promise<void> {
  await page.goto('/');
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
  await sidebarSettingsButton(page).click();
  await expect(page.getByTestId('settings-drawer')).toBeVisible({ timeout: 5000 });
  const section = page.getByTestId('alert-escalation-section');
  await section.getByRole('button', { expanded: false }).first().click();
  await expect(page.getByTestId('escalation-save')).toBeVisible();
}

async function save(page: Page): Promise<void> {
  const put = page.waitForResponse(
    (r) => r.request().method() === 'PUT' && r.url().endsWith('/api/v1/settings'),
  );
  await page.getByTestId('escalation-save').click();
  await put;
}

test.describe('Alert escalation', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('an operator adds, edits and removes a two-stage ladder', async ({ page }) => {
    await openEscalation(page);
    await expect(page.getByTestId('escalation-empty')).toBeVisible();

    // A ladder with no rule could not run: the server says so, nothing is stored.
    await page.getByTestId('escalation-add').click();
    await save(page);
    await expect(page.getByTestId('escalation-error')).toContainText('rule is required');

    await page.getByTestId('escalation-rule-0').fill('iface.down');
    await page.getByTestId('escalation-add-stage-0').click();
    await page.getByTestId('escalation-channel-0-1-syslog').check();
    await save(page);
    await expect(page.getByTestId('escalation-error')).toHaveCount(0);

    await page.reload();
    await openEscalation(page);
    await expect(page.getByTestId('escalation-rule-0')).toHaveValue('iface.down');
    await expect(page.getByTestId('escalation-after-0-0')).toHaveValue('5');
    await expect(page.getByTestId('escalation-after-0-1')).toHaveValue('10');
    await expect(page.getByTestId('escalation-channel-0-1-syslog')).toBeChecked();

    // Edit: stage 2 moves to 30 minutes and repeats hourly.
    await page.getByTestId('escalation-after-0-1').fill('30');
    await page.getByTestId('escalation-repeat-0').fill('60');
    await save(page);

    await page.reload();
    await openEscalation(page);
    await expect(page.getByTestId('escalation-after-0-1')).toHaveValue('30');
    await expect(page.getByTestId('escalation-repeat-0')).toHaveValue('60');

    await page.getByTestId('escalation-remove-0').click();
    await save(page);

    await page.reload();
    await openEscalation(page);
    await expect(page.getByTestId('escalation-empty')).toBeVisible();
  });

  test('an escalated alert shows its stage in the inbox', async ({ page }) => {
    await page.route(/\/api\/v1\/alerts(\?.*)?$/, (route) =>
      route.fulfill({
        json: {
          count: 1,
          alerts: [
            {
              id: 41,
              type: 'interface',
              severity: 'critical',
              title: 'Uplink Gi1/0/48 down',
              message: '',
              source: 'snmp',
              acknowledged: false,
              resolved: false,
              createdAt: '2026-10-07T10:00:00Z',
              metadata: {},
              rule: 'iface.down',
              escalationStage: 2,
              escalatedAt: '2026-10-07T10:30:00Z',
            },
          ],
        },
      }),
    );
    await page.goto('/alerts');

    await page.getByTestId('alert-row-41').click();
    await expect(page.getByTestId('alert-escalation')).toContainText('stage 2');
  });
});
