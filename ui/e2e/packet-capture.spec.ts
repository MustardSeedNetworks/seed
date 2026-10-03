import { readFile } from 'node:fs/promises';
import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Packet capture card on /network (#326, #239), against the E2E daemon.
 *
 * The daemon runs on lo, so the capture records the browser's own traffic to
 * it and the summary has something real to tally.
 *
 * scripts/build-backend-e2e.sh builds the CI daemon CGO_ENABLED=0, whose
 * capture adapter refuses every interface. The capture test skips on exactly
 * that refusal and on nothing else, so a capture-capable daemon (a libpcap
 * build with CAP_NET_RAW) runs it in full and any other failure fails it.
 */

const CGO_FREE_REFUSAL = 'built without CGO/libpcap';
// pcapgo writes the classic microsecond header, little-endian.
const PCAP_MAGIC = 'd4c3b2a1';

async function openNetwork(page: Page): Promise<void> {
  await skipSetupWizard(page);
  await page.goto('/network');
  await expect(page.getByTestId('packet-capture')).toBeVisible({ timeout: 10000 });
}

test.describe('Packet capture card', () => {
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

    await openNetwork(page);

    await expect(page.getByTestId('packet-capture-read-only')).toBeVisible();
    await expect(page.getByTestId('packet-capture-start')).toHaveCount(0);
    await expect(page.getByTestId('packet-capture-interface')).toHaveCount(0);
    expect(jobPosts).toEqual([]);
  });

  test('a capture on an interface that does not exist fails visibly', async ({ page }) => {
    await openNetwork(page);

    await page.getByTestId('packet-capture-interface').fill('seed-e2e-none0');
    await page.getByTestId('packet-capture-duration').fill('5');
    await page.getByTestId('packet-capture-start').click();

    await expect(page.getByTestId('packet-capture')).toHaveAttribute('data-phase', 'failed');
    await expect(page.getByTestId('packet-capture-failed')).toBeVisible();
    await expect(page.getByTestId('packet-capture-error-detail')).not.toBeEmpty();
    await expect(page.getByTestId('packet-capture-result')).toHaveCount(0);
    await expect(page.getByTestId('packet-capture-start')).toBeEnabled();
  });

  test('an operator captures on lo, stops, reads the summary and downloads the file', async ({
    page,
  }) => {
    await openNetwork(page);
    const card = page.getByTestId('packet-capture');

    await page.getByTestId('packet-capture-interface').fill('lo');
    await page.getByTestId('packet-capture-duration').fill('60');
    await page.getByTestId('packet-capture-start').click();

    await expect(card).toHaveAttribute('data-phase', /running|failed/);
    if ((await card.getAttribute('data-phase')) === 'failed') {
      const detail = (await page.getByTestId('packet-capture-error-detail').textContent()) ?? '';
      test.skip(
        detail.includes(CGO_FREE_REFUSAL),
        'this daemon is the CGO-free E2E build, which cannot capture',
      );
      throw new Error(`capture on lo failed: ${detail}`);
    }

    await expect(page.getByTestId('packet-capture-running')).toContainText('lo');
    // Traffic for the capture to record: the daemon's own HTTPS on lo.
    for (let i = 0; i < 5; i++) {
      await page.request.get('/__version');
    }
    await page.getByTestId('packet-capture-stop').click();

    const result = page.getByTestId('packet-capture-result');
    await expect(result).toHaveAttribute('data-stop-reason', 'stopped', { timeout: 15000 });
    await expect(page.getByTestId('packet-capture-protocols')).toContainText('TCP');
    await expect(page.getByTestId('packet-capture-talkers')).toContainText('127.0.0.1');
    await expect(page.getByTestId('packet-capture-empty')).toHaveCount(0);

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByTestId('packet-capture-download').click(),
    ]);
    expect(download.suggestedFilename()).toMatch(/^seed-capture-.+\.pcap$/);
    const file = await readFile(await download.path());
    expect(file.subarray(0, 4).toString('hex')).toBe(PCAP_MAGIC);
    expect(file.length).toBeGreaterThan(24);
  });
});
