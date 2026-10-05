import { createSocket } from 'node:dgram';
import { readdir, readFile } from 'node:fs/promises';
import { networkInterfaces } from 'node:os';
import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Multicast listen card on /network (#399), against the E2E daemon.
 *
 * The daemon and this runner share a host, so the live test sends real
 * datagrams to an organisation-local group out of a multicast-capable
 * interface, and multicast loopback delivers them to the daemon's membership
 * on the same interface. lo cannot join a group on Linux (no MULTICAST flag),
 * which is why the test looks for a real interface rather than using lo.
 *
 * POST /jobs shares the daemon's 5-per-minute bucket with every other
 * rate-limited route, so the spec starts one job. How a refused or failed
 * listen reads is covered by MulticastListenCard.test.tsx.
 */

const GROUP = '239.255.83.69';
const PORT = 45399;
// <linux/if.h>: an interface the listen can join must be up and multicast.
const IFF_UP = 0x1;
const IFF_MULTICAST = 0x1000;

interface MulticastInterface {
  name: string;
  address: string;
}

/** The first up, multicast-capable interface with an IPv4 address (Linux). */
async function multicastInterface(): Promise<MulticastInterface | null> {
  const addresses = networkInterfaces();
  let names: string[];
  try {
    names = await readdir('/sys/class/net');
  } catch {
    return null;
  }
  for (const name of names.sort()) {
    const flags = Number.parseInt(await readFile(`/sys/class/net/${name}/flags`, 'utf8'), 16);
    const required = IFF_UP | IFF_MULTICAST;
    const v4 = addresses[name]?.find((a) => a.family === 'IPv4' && !a.internal);
    if ((flags & required) === required && v4) {
      return { name, address: v4.address };
    }
  }
  return null;
}

async function openNetwork(page: Page): Promise<void> {
  await skipSetupWizard(page);
  await page.goto('/network');
  await expect(page.getByTestId('multicast-listen')).toBeVisible({ timeout: 10000 });
}

async function fillListen(page: Page, iface: string, seconds: string): Promise<void> {
  await page.getByTestId('multicast-listen-group').fill(GROUP);
  await page.getByTestId('multicast-listen-port').fill(String(PORT));
  await page.getByTestId('multicast-listen-interface').fill(iface);
  await page.getByTestId('multicast-listen-duration').fill(seconds);
}

test.describe('Multicast listen card', () => {
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

    await expect(page.getByTestId('multicast-listen-read-only')).toBeVisible();
    await expect(page.getByTestId('multicast-listen-start')).toHaveCount(0);
    await expect(page.getByTestId('multicast-listen-group')).toHaveCount(0);
    expect(jobPosts).toEqual([]);
  });

  test('an operator hears a stream, stops, and reads its sender', async ({ page }) => {
    const iface = await multicastInterface();
    test.skip(iface === null, 'no up, multicast-capable IPv4 interface on this host');
    if (iface === null) {
      return;
    }

    await openNetwork(page);
    await fillListen(page, iface.name, '60');
    const [start] = await Promise.all([
      page.waitForResponse(
        (response) =>
          response.request().method() === 'POST' && response.url().endsWith('/api/v1/jobs'),
      ),
      page.getByTestId('multicast-listen-start').click(),
    ]);
    expect(start.ok(), `job start answered ${start.status()}`).toBe(true);
    await expect(page.getByTestId('multicast-listen-running')).toContainText(iface.name);

    const sender = createSocket('udp4');
    await new Promise<void>((resolve) => sender.bind(0, iface.address, resolve));
    sender.setMulticastInterface(iface.address);
    sender.setMulticastLoopback(true);
    // The card reads "running" from the job's acceptance, before the daemon
    // has joined, so keep sending across the join rather than once.
    try {
      for (let i = 0; i < 30; i++) {
        await new Promise<void>((resolve, reject) =>
          sender.send(`seed-e2e ${i}`, PORT, GROUP, (err) => (err ? reject(err) : resolve())),
        );
        await page.waitForTimeout(100);
      }
    } finally {
      sender.close();
    }
    await page.getByTestId('multicast-listen-stop').click();

    await expect(page.getByTestId('multicast-listen-result')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('multicast-listen')).toHaveAttribute('data-phase', 'finished');
    await expect(page.getByTestId('multicast-listen-sources')).toContainText(iface.address);
    await expect(page.getByTestId('multicast-listen-totals')).toContainText(
      `${GROUP} port ${PORT} on ${iface.name}`,
    );
    await expect(page.getByTestId('multicast-listen-silent')).toHaveCount(0);
  });
});
