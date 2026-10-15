import { readFile } from 'node:fs/promises';
import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Packet capture card on /network (#326, #239), against the E2E daemon.
 *
 * The daemon runs on lo, so the capture records the browser's own traffic to
 * it and the summary has something real to tally.
 *
 * The daemon must be able to capture: a libpcap build with CAP_NET_RAW. CI's
 * is (the e2e-backend job, #2986), so a daemon that cannot capture fails the
 * capture test rather than skipping it.
 *
 * POST /jobs shares the daemon's 5-per-minute bucket with every other
 * rate-limited route, so the spec starts one job. How a capture that cannot
 * open its interface fails is covered by TestRunFailsWithoutLeavingAFile and
 * PacketCaptureCard.test.tsx.
 */

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

  test('an operator captures on lo, stops, reads the summary and downloads the file', async ({
    page,
  }) => {
    await openNetwork(page);
    const card = page.getByTestId('packet-capture');

    await page.getByTestId('packet-capture-interface').fill('lo');
    await page.getByTestId('packet-capture-duration').fill('60');
    await page.getByTestId('packet-capture-start').click();
    await expect(page.getByTestId('packet-capture-running')).toContainText('lo');
    // Traffic for the capture to record: the daemon's own HTTPS on lo.
    for (let i = 0; i < 5; i++) {
      await page.request.get('/__version');
    }
    await page.getByTestId('packet-capture-stop').click();

    // "running" comes from the job's acceptance, before the daemon opens the
    // interface, so only the capture's end says whether lo opened.
    await expect(card).toHaveAttribute('data-phase', /finished|failed/, { timeout: 15000 });
    if ((await card.getAttribute('data-phase')) === 'failed') {
      const detail = (await page.getByTestId('packet-capture-error-detail').textContent()) ?? '';
      throw new Error(`capture on lo failed: ${detail}`);
    }

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
