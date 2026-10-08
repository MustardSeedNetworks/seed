import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Scheduled reports card on /reports (#3209), against the E2E daemon.
 *
 * Schedules are Pro (`scheduled_reports`). The suite's daemon is unlicensed
 * (seed#2688), so these tests show the page by intercepting GET
 * /api/v1/license, and the daemon answers for itself:
 * - unlicensed, a save reaches POST /reports/schedules and is refused, and the
 *   editor stays open with the reason — the gate is not only the page's;
 * - on a licensed daemon (a trial started in its HOME), an operator creates,
 *   edits and deletes a schedule, and each reload reads back what was stored.
 * Each of the last two skips on the other licence state and on nothing else.
 */

const PRO_LICENCE = {
  tier: 'Pro',
  tierValue: 2,
  isTrialMode: false,
  canMintTokens: true,
  activated: true,
  features: ['export_csv_json', 'scheduled_reports'],
};

async function daemonHasSchedules(page: Page): Promise<boolean> {
  const response = await page.request.get('/api/v1/license');
  expect(response.ok(), `licence answered ${response.status()}`).toBe(true);
  const { features } = (await response.json()) as { features?: string[] };
  return features?.includes('scheduled_reports') ?? false;
}

async function openReports(page: Page, stubLicence: boolean): Promise<void> {
  if (stubLicence) {
    await page.route('**/api/v1/license', (route) => route.fulfill({ json: PRO_LICENCE }));
  }
  await page.goto('/reports');
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
}

async function save(page: Page): Promise<void> {
  const write = page.waitForResponse(
    (r) =>
      ['POST', 'PUT'].includes(r.request().method()) &&
      r.url().includes('/api/v1/reports/schedules'),
  );
  await page.getByTestId('schedule-save').click();
  await write;
}

test.describe('Scheduled reports', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('a viewer sees no way to change a schedule', async ({ page }) => {
    await page.route('**/api/v1/users/me', (route) =>
      route.fulfill({ json: { username: 'reader', role: 'viewer', isActive: true } }),
    );
    await page.route('**/api/v1/reports/schedules', (route) =>
      route.fulfill({
        json: {
          schedules: [
            {
              id: 's1',
              name: 'Monday summary',
              template: 'executive',
              format: 'pdf',
              schedule: { frequency: 'weekly', dayOfWeek: 1, hour: 6, minute: 0, timezone: 'UTC' },
              enabled: true,
              createdAt: '2026-10-08T00:00:00Z',
              updatedAt: '2026-10-08T00:00:00Z',
            },
          ],
        },
      }),
    );
    await openReports(page, true);

    await expect(page.getByTestId('schedule-row')).toContainText('Monday at 06:00 (UTC)');
    await expect(page.getByTestId('schedule-new')).toHaveCount(0);
    await expect(page.getByTestId('schedule-edit-s1')).toHaveCount(0);
    await expect(page.getByTestId('schedule-delete-s1')).toHaveCount(0);
  });

  test('an unlicensed daemon refuses the save itself', async ({ page }) => {
    test.skip(await daemonHasSchedules(page), 'the daemon is licensed');
    await openReports(page, true);

    await page.getByTestId('schedule-new').click();
    await page.getByTestId('schedule-name').fill('Weekly summary');
    await save(page);
    await expect(page.getByTestId('schedule-error')).toBeVisible();
    await expect(page.getByTestId('schedule-editor')).toBeVisible();
  });

  test('an operator creates, edits and deletes a schedule', async ({ page }) => {
    test.skip(!(await daemonHasSchedules(page)), 'needs a licensed daemon');
    // Three loads of /reports; webkit on CI is slow to settle each.
    test.slow();
    await openReports(page, false);
    await expect(page.getByTestId('schedules-empty')).toBeVisible();

    // A time zone the daemon cannot load is refused with the field named.
    await page.getByTestId('schedule-new').click();
    await page.getByTestId('schedule-name').fill('Month end');
    await page.getByTestId('schedule-frequency').selectOption('monthly');
    await page.getByTestId('schedule-day-of-month').selectOption('28');
    await page.getByTestId('schedule-time').fill('07:30');
    await page.getByTestId('schedule-timezone').fill('Mars/Olympus');
    await save(page);
    await expect(page.getByTestId('schedule-error')).toContainText('unknown timezone');

    await page.getByTestId('schedule-timezone').fill('UTC');
    await save(page);
    await expect(page.getByTestId('schedule-editor')).toHaveCount(0);

    await openReports(page, false);
    const row = page.getByTestId('schedule-row');
    await expect(row).toHaveCount(1);
    await expect(row).toContainText('Month end');
    await expect(row).toContainText('Day 28 at 07:30 (UTC)');

    // Edit: weekly on Friday, a CSV inventory, paused.
    await row.getByRole('button', { name: 'Edit' }).click();
    await page.getByTestId('schedule-template').selectOption('inventory');
    await page.getByTestId('schedule-format').selectOption('csv');
    await page.getByTestId('schedule-frequency').selectOption('weekly');
    await page.getByTestId('schedule-day-of-week').selectOption('5');
    await page.getByTestId('schedule-enabled').uncheck();
    await save(page);
    await expect(page.getByTestId('schedule-editor')).toHaveCount(0);

    await openReports(page, false);
    await expect(row).toContainText('Device inventory · CSV');
    await expect(row).toContainText('Friday at 07:30 (UTC)');
    await expect(row).toContainText('Paused');

    // The suite's daemon is shared: leave no schedule behind.
    const remove = page.waitForResponse(
      (r) => r.request().method() === 'DELETE' && r.url().includes('/api/v1/reports/schedules/'),
    );
    await row.getByRole('button', { name: 'Delete' }).click();
    await remove;
    await expect(page.getByTestId('schedules-empty')).toBeVisible();
  });
});
