import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * DSCP check card on /performance (#400), against the E2E daemon.
 *
 * The check is sold as `dscp_verification` (Pro), and POST /jobs enforces it
 * for each qos kind. The suite's daemon is unlicensed (seed#2688), so:
 * - below Pro the card shows the upgrade prompt and sends nothing;
 * - with the licence stubbed, the start reaches POST /jobs and comes back
 *   402, so the gate is not only the card's;
 * - on a licensed daemon (a trial started in its HOME), one page listens and
 *   a second page sends to it over loopback, which is the two-Seed check
 *   with both ends on one daemon, and the listener reads every class kept.
 * POST /jobs shares the daemon's 5-per-minute bucket, so each test starts at
 * most two jobs. Per-class verdicts from fixtures are DscpCheckCard.test.tsx.
 */

const PORT = '45401';

const PRO_LICENCE = {
  tier: 'Pro',
  tierValue: 2,
  isTrialMode: false,
  canMintTokens: true,
  activated: true,
  features: ['dscp_verification'],
};

const FREE_LICENCE = {
  tier: 'Free',
  tierValue: 0,
  isTrialMode: false,
  canMintTokens: false,
  activated: false,
  features: [],
};

async function daemonHasDscp(page: Page): Promise<boolean> {
  const response = await page.request.get('/api/v1/license');
  expect(response.ok(), `licence answered ${response.status()}`).toBe(true);
  const { features } = (await response.json()) as { features?: string[] };
  return features?.includes('dscp_verification') ?? false;
}

async function openPerformance(page: Page, licence?: object): Promise<void> {
  await skipSetupWizard(page);
  if (licence) {
    await page.route('**/api/v1/license', (route) => route.fulfill({ json: licence }));
  }
  await page.goto('/performance');
  await expect(page.getByTestId('dscp-check')).toBeVisible({ timeout: 10000 });
}

function recordJobPosts(page: Page): string[] {
  const posts: string[] = [];
  page.on('request', (request) => {
    if (request.method() === 'POST' && request.url().includes('/api/v1/jobs')) {
      posts.push(request.url());
    }
  });
  return posts;
}

test.describe('DSCP check card', () => {
  test('below Pro the card offers the upgrade and sends nothing', async ({ page }) => {
    const jobPosts = recordJobPosts(page);
    await openPerformance(page, FREE_LICENCE);

    await expect(page.getByTestId('dscp-check-upgrade')).toContainText('Pro tier');
    await expect(page.getByTestId('dscp-check-start')).toHaveCount(0);
    await expect(page.getByTestId('dscp-check-port')).toHaveCount(0);
    expect(jobPosts).toEqual([]);
  });

  test('a viewer sees the reason and no start control', async ({ page }) => {
    await page.route('**/api/v1/users/me', (route) =>
      route.fulfill({ json: { username: 'reader', role: 'viewer', isActive: true } }),
    );
    const jobPosts = recordJobPosts(page);
    await openPerformance(page, PRO_LICENCE);

    await expect(page.getByTestId('dscp-check-read-only')).toBeVisible();
    await expect(page.getByTestId('dscp-check-start')).toHaveCount(0);
    expect(jobPosts).toEqual([]);
  });

  test('an unlicensed daemon refuses the check at the job', async ({ page }) => {
    test.skip(await daemonHasDscp(page), 'the daemon is licensed for dscp_verification');
    await openPerformance(page, PRO_LICENCE);

    const [start] = await Promise.all([
      page.waitForResponse(
        (response) =>
          response.request().method() === 'POST' && response.url().endsWith('/api/v1/jobs'),
      ),
      page.getByTestId('dscp-check-start').click(),
    ]);
    expect(start.status()).toBe(402);
    await expect(page.getByTestId('dscp-check')).toHaveAttribute('data-phase', 'failed');
    await expect(page.getByTestId('dscp-check-result')).toHaveCount(0);
  });

  test('one page listens, another sends over loopback, and every class is kept', async ({
    page,
    context,
  }) => {
    test.skip(
      !(await daemonHasDscp(page)),
      'the daemon is unlicensed; start a trial in its HOME to run this',
    );
    await openPerformance(page);
    const listener = page.getByTestId('dscp-check');

    await page.getByTestId('dscp-check-port').fill(PORT);
    await page.getByTestId('dscp-check-duration').fill('20');
    await page.getByTestId('dscp-check-start').click();
    await expect(listener).toHaveAttribute('data-phase', 'running');

    const sender = await context.newPage();
    await openPerformance(sender);
    await sender.getByTestId('dscp-check-mode-send').click();
    await sender.getByTestId('dscp-check-target').fill('127.0.0.1');
    await sender.getByTestId('dscp-check-port').fill(PORT);
    await sender.getByTestId('dscp-check-start').click();
    await expect(sender.getByTestId('dscp-check-sent')).toContainText('127.0.0.1 port 45401', {
      timeout: 15000,
    });

    // A stopped listen keeps what it heard, so the verdict needs no wait for
    // the window to close.
    await page.getByTestId('dscp-check-stop').click();
    await expect(listener).toHaveAttribute('data-phase', 'finished', { timeout: 10000 });
    await expect(page.getByTestId('dscp-check-run')).toContainText('127.0.0.1');
    await expect(page.getByTestId('dscp-check-verdict')).toHaveAttribute('data-pass', 'true');
    await expect(
      page.getByTestId('dscp-check-classes').getByRole('row').filter({ hasText: 'EF (46)' }),
    ).toContainText('Kept');
  });
});
