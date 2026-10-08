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

function historyBody(range: string): Record<string, unknown> {
  const to = Date.parse('2026-10-08T12:00:00Z');
  const points = [0, 1, 2, 3].map((i) => ({
    sampledAt: new Date(to - (4 - i) * 60_000).toISOString(),
    inOctetsPerSec: 812_500_000,
    outOctetsPerSec: 93_750_000,
    inErrorsPerSec: i === 2 ? 4.2 : 0,
    outErrorsPerSec: 0,
    inDiscardsPerSec: 0,
    outDiscardsPerSec: 0,
  }));
  return {
    targetId: 'target-core',
    ifIndex: 3,
    range,
    from: new Date(to - 3_600_000).toISOString(),
    to: new Date(to).toISOString(),
    bucketSeconds: 15,
    points: range === '7d' ? [] : points,
  };
}

test('opens an interface to its traffic and error history', async ({ page }) => {
  await page.route('**/api/v1/topology/interfaces', (route) =>
    route.fulfill({ json: { interfaces: INTERFACES, count: INTERFACES.length } }),
  );
  const ranges: string[] = [];
  await page.route('**/api/v1/topology/interfaces/history?*', (route) => {
    const url = new URL(route.request().url());
    const range = url.searchParams.get('range') ?? '';
    ranges.push(`${url.searchParams.get('target')}/${url.searchParams.get('ifIndex')}/${range}`);
    return route.fulfill({ json: historyBody(range) });
  });
  await skipSetupWizard(page);
  await page.goto('/interfaces');
  await expect(page.getByTestId('interfaces-table')).toBeVisible({ timeout: 10000 });

  await page.getByRole('button', { name: 'TenGigabitEthernet1/0/3' }).click();
  const history = page.getByTestId('interface-history');
  await expect(history.getByRole('heading', { name: 'TenGigabitEthernet1/0/3' })).toBeFocused();
  await expect(history.getByTestId('interface-history-traffic-peaks')).toHaveText(
    'Peak In 6.5 Gbps · Peak Out 750 Mbps',
  );
  await expect(history.getByTestId('interface-history-errors-peaks')).toHaveText(
    'Peak Errors 4.2/s · Peak Discards 0/s',
  );

  await history.getByTestId('interface-history-range').selectOption('7d');
  await expect(history.getByTestId('interface-history-empty')).toBeVisible();
  expect(ranges).toEqual(['target-core/3/24h', 'target-core/3/7d']);

  await history.getByTestId('interface-history-close').click();
  await expect(history).toHaveCount(0);
});

test('the interface history fits a 390px phone', async ({ page }) => {
  await page.route('**/api/v1/topology/interfaces', (route) =>
    route.fulfill({ json: { interfaces: INTERFACES, count: INTERFACES.length } }),
  );
  await page.route('**/api/v1/topology/interfaces/history?*', (route) =>
    route.fulfill({ json: historyBody('24h') }),
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await skipSetupWizard(page);
  await page.goto('/interfaces');
  await page.getByRole('button', { name: 'TenGigabitEthernet1/0/3' }).click();
  await expect(page.getByTestId('interface-history-traffic')).toBeVisible({ timeout: 10000 });

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - 390);
  expect(overflow).toBeLessThanOrEqual(0);
});
