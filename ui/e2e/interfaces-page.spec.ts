import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Interfaces page (UI-SEED-21): the estate's interfaces sorted by error rate.
 *
 * The empty state runs against the real daemon, which the suite starts with no
 * polling targets. The sorted list is mocked: a test host has no SNMP agent to
 * poll, and the order only means something with rates in it.
 */

function iface(
  ifIndex: number,
  name: string,
  inErrorsPerSec: number | null,
): Record<string, unknown> {
  return {
    targetId: 'target-core',
    targetName: 'core-sw-01.example.net',
    ifIndex,
    name,
    alias: 'uplink to distribution block B',
    operStatus: inErrorsPerSec === null ? 'down' : 'up',
    speedBps: 10_000_000_000,
    ...(inErrorsPerSec === null
      ? {}
      : {
          rates: {
            sampledAt: '2026-10-07T12:00:00Z',
            inOctetsPerSec: 812_500_000,
            outOctetsPerSec: 93_750_000,
            inUtilizationPct: 65,
            outUtilizationPct: 7.5,
            inErrorsPerSec,
            outErrorsPerSec: 0,
            inDiscardsPerSec: 0.004,
            outDiscardsPerSec: 0,
          },
        }),
  };
}

const INTERFACES = [
  iface(3, 'TenGigabitEthernet1/0/3', 4.2),
  iface(1, 'TenGigabitEthernet1/0/1', 0.5),
  iface(2, 'TenGigabitEthernet1/0/2', 0),
  iface(4, 'TenGigabitEthernet1/0/4', null),
];

async function names(page: Page): Promise<string[]> {
  return page.getByTestId('interface-row').getByTestId('interface-name').allTextContents();
}

test('an estate with no polling targets points at adding one', async ({ page }) => {
  await skipSetupWizard(page);
  await page.goto('/interfaces');
  await expect(page.getByTestId('interfaces-empty')).toBeVisible({ timeout: 10000 });
  await page.getByTestId('interfaces-empty').getByRole('link').click();
  await expect(page).toHaveURL(/\/polling-targets$/);
});

test('sorts interfaces by error rate and keeps unrated ones last', async ({ page }) => {
  await page.route('**/api/v1/topology/interfaces', (route) =>
    route.fulfill({ json: { interfaces: INTERFACES, count: INTERFACES.length } }),
  );
  await skipSetupWizard(page);
  await page.goto('/interfaces');
  await expect(page.getByTestId('interfaces-table')).toBeVisible({ timeout: 10000 });

  await expect
    .poll(() => names(page))
    .toEqual([
      'TenGigabitEthernet1/0/3',
      'TenGigabitEthernet1/0/1',
      'TenGigabitEthernet1/0/2',
      'TenGigabitEthernet1/0/4',
    ]);

  await page.getByTestId('interfaces-sort-errors').click();
  await expect
    .poll(() => names(page))
    .toEqual([
      'TenGigabitEthernet1/0/2',
      'TenGigabitEthernet1/0/1',
      'TenGigabitEthernet1/0/3',
      'TenGigabitEthernet1/0/4',
    ]);
  await expect(page.getByTestId('interfaces-no-data')).toHaveCount(0);
});

test('the interface table fits a 390px phone', async ({ page }) => {
  await page.route('**/api/v1/topology/interfaces', (route) =>
    route.fulfill({ json: { interfaces: INTERFACES, count: INTERFACES.length } }),
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await skipSetupWizard(page);
  await page.goto('/interfaces');
  const table = page.getByTestId('interfaces-table');
  await expect(table).toBeVisible({ timeout: 10000 });

  const overflow = await table.evaluate((el) => {
    const box = el.parentElement as HTMLElement;
    return Math.max(el.scrollWidth - box.clientWidth, document.documentElement.scrollWidth - 390);
  });
  expect(overflow).toBeLessThanOrEqual(0);
});
