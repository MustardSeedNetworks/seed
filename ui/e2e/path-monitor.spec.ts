import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Path monitor card on /path (#165), against the E2E daemon.
 *
 * Path monitoring is sold with Path Analysis (Pro), and the job kind carries
 * the gate as well as the page. The suite's daemon is unlicensed (seed#2688),
 * so these tests show the page by intercepting GET /api/v1/license, and the
 * daemon answers for itself:
 * - unlicensed, the start reaches POST /jobs and comes back 402 — the gate is
 *   not only the page's;
 * - on a licensed daemon (a trial started in its HOME), the monitor traces
 *   127.0.0.1 over lo, the hop table fills from the live event stream, and a
 *   stop keeps it.
 * Each of the last two skips on the other licence state and on nothing else.
 */

const PRO_LICENCE = {
  tier: 'Pro',
  tierValue: 2,
  isTrialMode: false,
  canMintTokens: true,
  activated: true,
  features: ['path_analysis'],
};

async function daemonHasPathAnalysis(page: Page): Promise<boolean> {
  const response = await page.request.get('/api/v1/license');
  expect(response.ok(), `licence answered ${response.status()}`).toBe(true);
  const { features } = (await response.json()) as { features?: string[] };
  return features?.includes('path_analysis') ?? false;
}

async function openPath(page: Page, stubLicence: boolean): Promise<void> {
  await skipSetupWizard(page);
  if (stubLicence) {
    await page.route('**/api/v1/license', (route) => route.fulfill({ json: PRO_LICENCE }));
  }
  await page.goto('/path');
  await expect(page.getByTestId('path-monitor')).toBeVisible({ timeout: 10000 });
}

test.describe('Path monitor card', () => {
  test('a viewer sees the reason and no start control', async ({ page }) => {
    await page.route('**/api/v1/users/me', (route) =>
      route.fulfill({ json: { username: 'reader', role: 'viewer', isActive: true } }),
    );
    const jobPosts: string[] = [];
    page.on('request', (request) => {
      if (request.method() === 'POST' && request.url().includes('/api/v1/jobs')) {
        jobPosts.push(request.url());
      }
    });

    await openPath(page, true);

    await expect(page.getByTestId('path-monitor-read-only')).toBeVisible();
    await expect(page.getByTestId('path-monitor-start')).toHaveCount(0);
    await expect(page.getByTestId('path-monitor-target')).toHaveCount(0);
    expect(jobPosts).toEqual([]);
  });

  test('an unlicensed daemon refuses the monitor at the job', async ({ page }) => {
    test.skip(await daemonHasPathAnalysis(page), 'the daemon is licensed for path_analysis');
    await openPath(page, true);

    await page.getByTestId('path-monitor-target').fill('127.0.0.1');
    const [start] = await Promise.all([
      page.waitForResponse(
        (response) =>
          response.request().method() === 'POST' && response.url().endsWith('/api/v1/jobs'),
      ),
      page.getByTestId('path-monitor-start').click(),
    ]);
    expect(start.status()).toBe(402);

    await expect(page.getByTestId('path-monitor')).toHaveAttribute('data-phase', 'failed');
    await expect(page.getByTestId('path-monitor-failed')).toBeVisible();
    await expect(page.getByTestId('path-monitor-hops')).toHaveCount(0);
  });

  test('an operator monitors loopback, watches the rounds climb, and stops', async ({ page }) => {
    test.skip(
      !(await daemonHasPathAnalysis(page)),
      'the daemon is unlicensed; start a trial in its HOME to run this',
    );
    await openPath(page, false);
    const card = page.getByTestId('path-monitor');

    await page.getByTestId('path-monitor-target').fill('127.0.0.1');
    await page.getByTestId('path-monitor-start').click();
    await expect(card).toHaveAttribute('data-phase', 'running');

    // Rounds are a second apart: a count past one proves the table is fed by
    // the stream, not by a single answer.
    await expect(page.getByTestId('path-monitor-rounds')).toHaveText(
      /^[2-9]\d* rounds to 127\.0\.0\.1$/,
      {
        timeout: 15000,
      },
    );
    const hop = page.getByTestId('path-monitor-hop').first();
    await expect(hop).toHaveAttribute('data-ttl', '1');
    await expect(hop.getByTestId('path-monitor-hop-address')).toHaveText(
      /^(127\.0\.0\.1|localhost)$/,
    );
    await expect(hop.getByTestId('path-monitor-hop-loss')).toHaveText('0%');

    await page.getByTestId('path-monitor-stop').click();
    await expect(card).toHaveAttribute('data-phase', 'stopped', { timeout: 10000 });
    await expect(page.getByTestId('path-monitor-hop')).not.toHaveCount(0);
    await expect(page.getByTestId('path-monitor-start')).toBeEnabled();
  });
});
