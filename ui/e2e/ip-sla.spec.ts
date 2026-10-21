import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * IP SLA (UI-SEED-48), against the E2E daemon.
 *
 * The daemon polls no Cisco target, so it lists no operation: the first test
 * reads that from the real API. The second routes the operations list to one
 * failing and one jitter operation, and routes the history read so the
 * window control can be seen asking the server for a new window.
 */

const OPERATIONS = {
  count: 2,
  operations: [
    {
      targetId: 't1',
      targetName: 'edge-rtr',
      observedAt: '2026-10-21T12:00:00Z',
      index: 20,
      tag: 'to-dc2',
      rttType: 1,
      target: '10.9.0.1',
      thresholdMs: 100,
      frequencySeconds: 60,
      status: 1,
      latest: { completed: false, sense: 4, rttMs: 0 },
    },
    {
      targetId: 't1',
      targetName: 'edge-rtr',
      observedAt: '2026-10-21T12:00:00Z',
      index: 10,
      rttType: 9,
      target: '10.9.0.5',
      thresholdMs: 50,
      frequencySeconds: 30,
      status: 1,
      latest: {
        completed: true,
        sense: 1,
        rttMs: 31,
        jitter: { avgJitterMs: 4, received: 100, lost: 0, mos: 4.3 },
      },
    },
  ],
};

test.describe('IP SLA', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('the daemon lists no operation and the page says so', async ({ page }) => {
    const list = page.waitForResponse((r) => r.url().endsWith('/api/v1/ip-sla/operations'));
    await page.goto('/ip-sla');
    expect((await list).status(), 'GET /api/v1/ip-sla/operations').toBe(200);
    await expect(page.getByTestId('ipsla-counts')).toHaveText('0 operations', { timeout: 10000 });
    await expect(page.getByText(/No IP SLA operation yet/)).toBeVisible();
  });

  test('an operation opens its trend and the window re-reads it', async ({ page }) => {
    await page.route('**/api/v1/ip-sla/operations', (route) => route.fulfill({ json: OPERATIONS }));
    const ranges: string[] = [];
    await page.route('**/api/v1/ip-sla/operations/history?*', (route) => {
      const url = new URL(route.request().url());
      ranges.push(`${url.searchParams.get('index')}:${url.searchParams.get('range')}`);
      return route.fulfill({
        json: {
          targetId: 't1',
          index: Number(url.searchParams.get('index')),
          range: url.searchParams.get('range'),
          from: '2026-10-20T12:00:00Z',
          to: '2026-10-21T12:00:00Z',
          bucketSeconds: 360,
          points: [
            {
              sampledAt: '2026-10-21T11:00:00Z',
              runs: 12,
              completedPct: 100,
              rttMs: 30,
              jitterMs: 3,
              lossPct: 0,
            },
            {
              sampledAt: '2026-10-21T11:06:00Z',
              runs: 12,
              completedPct: 100,
              rttMs: 31,
              jitterMs: 4,
              lossPct: 1,
            },
          ],
        },
      });
    });

    await page.goto('/ip-sla');
    await expect(page.getByTestId('ipsla-counts')).toHaveText('2 operations · 1 failing', {
      timeout: 10000,
    });
    await expect(page.getByTestId('ipsla-op-t1#20')).toContainText('Timed out');
    await expect(page.getByTestId('ipsla-history-empty')).toHaveCount(0);

    await page.getByTestId('ipsla-op-t1#10').click();
    await expect(page.getByTestId('ipsla-type')).toHaveText(/UDP jitter/);
    await expect(page.getByTestId('ipsla-history-timing-peaks')).toHaveText(
      'Peak RTT 31 ms · Peak Jitter 4 ms',
    );

    await page.getByTestId('ipsla-history-range').selectOption('7d');
    await expect.poll(() => ranges.at(-1)).toBe('10:7d');
    expect(ranges).toContain('20:24h');
  });
});
