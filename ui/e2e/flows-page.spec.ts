import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Flow explorer (UI-SEED-23): the top talkers, conversations and applications
 * over a window.
 *
 * The empty and clamped states run against the real daemon, which the suite
 * starts unlicensed and with no flow collector bound. The populated lists are
 * mocked: filling them needs a Pro collector and an exporter.
 */

const WINDOW = {
  from: '2026-10-06T00:00:00Z',
  to: '2026-10-07T00:00:00Z',
  days: 1,
  resolution: 'hourly',
  source: 'raw',
  clamped: false,
  requestedDays: 1,
};

test('a window with no flows says so, and a window past the licence is named', async ({ page }) => {
  await skipSetupWizard(page);
  await page.goto('/flows');
  await expect(page.getByTestId('flows-empty')).toBeVisible({ timeout: 10000 });
  await expect(page.getByTestId('flows-clamped')).toBeHidden();

  const reread = page.waitForResponse(
    (r) => r.url().includes('/api/v1/flows/top-talkers?range=90d&by=packets') && r.ok(),
  );
  await page.getByTestId('flows-rank').selectOption('packets');
  await page.getByTestId('flows-range').selectOption('90d');
  await reread;
  await expect(page.getByTestId('flows-clamped')).toBeVisible();
  await expect(page.getByTestId('flows-empty')).toBeVisible();
});

test('lists the top talkers, conversations and applications', async ({ page }) => {
  const seen: string[] = [];
  await page.route('**/api/v1/flows/top-*', (route) => {
    const url = new URL(route.request().url());
    seen.push(`${url.pathname}?${url.searchParams.toString()}`);
    const by = url.searchParams.get('by');
    const body: Record<string, unknown> = { window: WINDOW, by };
    if (url.pathname.endsWith('top-talkers')) {
      body.talkers = [
        { addr: '10.20.0.15', bytes: 734_003_200, packets: 512_000 },
        { addr: '2001:db8:20::42', bytes: 52_428_800, packets: 40_960 },
      ];
    } else if (url.pathname.endsWith('top-conversations')) {
      body.conversations = [
        {
          addrA: '10.20.0.15',
          addrB: '198.51.100.7',
          protocol: 6,
          bytes: 734_003_200,
          packets: 512_000,
        },
      ];
    } else {
      body.applications = [
        { name: 'https', bytes: 734_003_200, packets: 512_000 },
        { name: 'unknown', bytes: 1_048_576, packets: 900 },
      ];
    }
    return route.fulfill({ json: body });
  });
  await skipSetupWizard(page);
  await page.goto('/flows');

  await expect(page.getByTestId('flows-talkers-row')).toHaveCount(2, { timeout: 10000 });
  await expect(page.getByTestId('flows-talkers-row').first()).toContainText('10.20.0.15');
  await expect(page.getByTestId('flows-conversations-row')).toContainText(
    '10.20.0.15 ↔ 198.51.100.7',
  );
  await expect(page.getByTestId('flows-conversations-row')).toContainText('TCP');
  await expect(page.getByTestId('flows-applications-row').last()).toContainText('Unidentified');

  seen.length = 0;
  await page.getByTestId('flows-range').selectOption('7d');
  await expect.poll(() => seen.length).toBe(3);
  expect(seen.every((q) => q.endsWith('?range=7d&by=bytes'))).toBe(true);
});
