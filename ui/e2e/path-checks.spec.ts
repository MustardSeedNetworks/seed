import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Multi-path (#395) and path MTU (#435) cards on /path, against the E2E daemon.
 *
 * Both are sold with Path Analysis (Pro) and the job kinds carry the gate as
 * well as the page. The suite's daemon is unlicensed (seed#2688), so these
 * show the page by intercepting GET /api/v1/license and let the daemon answer:
 * unlicensed, a start comes back 402 from POST /jobs; on a licensed daemon
 * (a trial started in its HOME), each check runs over lo against 127.0.0.1.
 * Path MTU probes on a raw ICMP socket, so that run also needs the binary to
 * carry cap_net_raw, as a packaged install does (deploy/nfpm/postinstall.sh).
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

const CHECKS = ['multi-path', 'path-mtu'] as const;

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
  for (const check of CHECKS) {
    await expect(page.getByTestId(check)).toBeVisible({ timeout: 10000 });
  }
}

test.describe('Path checks', () => {
  test('a viewer sees the reason and no start control', async ({ page }) => {
    await page.route('**/api/v1/users/me', (route) =>
      route.fulfill({ json: { username: 'reader', role: 'viewer', isActive: true } }),
    );
    await openPath(page, true);

    for (const check of CHECKS) {
      await expect(page.getByTestId(`${check}-read-only`)).toBeVisible();
      await expect(page.getByTestId(`${check}-start`)).toHaveCount(0);
    }
  });

  // One POST, not one per card: /jobs shares a five-a-minute limiter across the
  // whole suite, and the 402 for both kinds is pinned in
  // TestPathDiscoveryKindsRequirePathAnalysis.
  test('an unlicensed daemon refuses the check at the job', async ({ page }) => {
    test.skip(await daemonHasPathAnalysis(page), 'the daemon is licensed for path_analysis');
    await openPath(page, true);

    await page.getByTestId('multi-path-target').fill('127.0.0.1');
    const [start] = await Promise.all([
      page.waitForResponse(
        (response) =>
          response.request().method() === 'POST' && response.url().endsWith('/api/v1/jobs'),
      ),
      page.getByTestId('multi-path-start').click(),
    ]);
    expect(start.status()).toBe(402);
    await expect(page.getByTestId('multi-path')).toHaveAttribute('data-phase', 'failed');
    await expect(page.getByTestId('multi-path-result')).toHaveCount(0);
  });

  test('an operator finds one route to loopback', async ({ page }) => {
    test.skip(
      !(await daemonHasPathAnalysis(page)),
      'the daemon is unlicensed; start a trial in its HOME to run this',
    );
    await openPath(page, false);

    await page.getByTestId('multi-path-target').fill('127.0.0.1');
    await page.getByTestId('multi-path-start').click();
    await expect(page.getByTestId('multi-path')).toHaveAttribute('data-phase', 'finished', {
      timeout: 30000,
    });
    await expect(page.getByTestId('multi-path-summary')).toHaveText(
      'One route to 127.0.0.1 in 8 traces.',
    );
    const route = page.getByTestId('multi-path-route');
    await expect(route).toHaveCount(1);
    await expect(route).toHaveAttribute('data-completed', 'true');
    await expect(route.getByTestId('multi-path-hop').last()).toContainText('127.0.0.1');
  });

  test('an operator measures the loopback path MTU', async ({ page }) => {
    test.skip(
      !(await daemonHasPathAnalysis(page)),
      'the daemon is unlicensed; start a trial in its HOME to run this',
    );
    await openPath(page, false);

    await page.getByTestId('path-mtu-target').fill('127.0.0.1');
    await page.getByTestId('path-mtu-start').click();
    await expect(page.getByTestId('path-mtu')).toHaveAttribute('data-phase', 'finished', {
      timeout: 30000,
    });
    // lo's MTU (65536) is above the 9000-byte search ceiling, which a
    // loopback probe always reaches.
    await expect(page.getByTestId('path-mtu-result')).toHaveAttribute('data-status', 'ok');
    await expect(page.getByTestId('path-mtu-value')).toHaveText('At least 9000');
    await expect(page.getByTestId('path-mtu-verdict')).toHaveText(
      'Every hop carries the largest size probed.',
    );
  });
});
