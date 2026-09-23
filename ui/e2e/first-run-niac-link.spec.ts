import { type APIRequestContext, expect, type Page, test } from '@playwright/test';

/**
 * A fresh install finds its neighbours with no Settings visit (seed#2674).
 *
 * Runs only under scripts/e2e-niac-link.sh, which puts a NIAC scenario on the
 * far end of a veth pair, starts seed on the near end with nothing but the
 * interface, port and paths configured, and exports:
 *
 *   SEED_E2E_NIAC_LINK     set to 1 to enable this spec
 *   SEED_E2E_STARTED_AT    epoch milliseconds just before seed was started
 *   SEED_E2E_EXPECTED_IPS  comma-separated addresses the scenario serves
 *
 * The 90 s budget is the row's acceptance and is measured from daemon start,
 * so the time global setup spends signing in counts against it.
 *
 * The daemon's inventory is checked before any page loads. Network and
 * Security POST a scan of their own on mount (seed#2691), and that scan alone
 * finds every device on an install whose discovery methods are all off, so an
 * assertion made only through the pages cannot tell a working default from a
 * broken one.
 */
const ACCEPTANCE_MS = 90_000;

const enabled = process.env.SEED_E2E_NIAC_LINK === '1';
const startedAt = Number(process.env.SEED_E2E_STARTED_AT);
const expectedIPs = (process.env.SEED_E2E_EXPECTED_IPS ?? '').split(',').filter(Boolean);

test.skip(!enabled, 'needs scripts/e2e-niac-link.sh (a NIAC scenario over a veth pair)');

function remainingBudget(): number {
  return Math.max(startedAt + ACCEPTANCE_MS - Date.now(), 1);
}

function listedAddresses(body: unknown): Set<string> {
  const { devices } = body as { devices?: unknown };
  if (!Array.isArray(devices)) {
    return new Set();
  }
  return new Set(
    devices
      .map((device: unknown) => (device as { ip?: unknown }).ip)
      .filter((ip): ip is string => typeof ip === 'string'),
  );
}

async function expectDaemonFoundEveryDevice(request: APIRequestContext): Promise<void> {
  await expect
    .poll(
      async () => {
        const response = await request.get('/api/v1/security/devices');
        const found = response.ok() ? listedAddresses(await response.json()) : new Set<string>();
        return expectedIPs.filter((ip) => !found.has(ip));
      },
      {
        message: 'addresses the daemon has not found',
        timeout: remainingBudget(),
        intervals: [1_000],
      },
    )
    .toEqual([]);
}

async function expectEveryDeviceListed(page: Page, route: string): Promise<void> {
  await expect(async () => {
    await page.goto(route);
    await page.getByTestId('discovery-card-maximize').click({ timeout: 2_000 });
    const dialog = page.getByRole('dialog');
    for (const ip of expectedIPs) {
      // A device with no name shows its address in the name column as well.
      await expect(dialog.getByText(ip, { exact: true }).first()).toBeVisible({ timeout: 1_000 });
    }
  }).toPass({ timeout: remainingBudget(), intervals: [2_000] });
}

test.describe('first run over a NIAC link', () => {
  test.setTimeout(ACCEPTANCE_MS + 30_000);

  test('Network and Security list every scenario device within 90 s of start', async ({ page }) => {
    expect(Number.isFinite(startedAt)).toBe(true);
    expect(expectedIPs.length).toBeGreaterThan(0);

    await expectDaemonFoundEveryDevice(page.request);
    await expectEveryDeviceListed(page, '/network');
    await expectEveryDeviceListed(page, '/security');

    const elapsed = Date.now() - startedAt;
    test.info().annotations.push({
      type: 'first-run discovery',
      description: `${expectedIPs.length} devices on both pages ${elapsed} ms after start`,
    });
    expect(elapsed).toBeLessThanOrEqual(ACCEPTANCE_MS);
  });
});
