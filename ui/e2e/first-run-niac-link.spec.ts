import { expect, type Page, test } from '@playwright/test';

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
 */
const ACCEPTANCE_MS = 90_000;

const enabled = process.env.SEED_E2E_NIAC_LINK === '1';
const startedAt = Number(process.env.SEED_E2E_STARTED_AT);
const expectedIPs = (process.env.SEED_E2E_EXPECTED_IPS ?? '').split(',').filter(Boolean);

test.skip(!enabled, 'needs scripts/e2e-niac-link.sh (a NIAC scenario over a veth pair)');

function remainingBudget(): number {
  return Math.max(startedAt + ACCEPTANCE_MS - Date.now(), 1);
}

async function expectEveryDeviceListed(page: Page, route: string): Promise<void> {
  await expect(async () => {
    await page.goto(route);
    await page.getByTestId('discovery-card-maximize').click({ timeout: 2_000 });
    const dialog = page.getByRole('dialog');
    for (const ip of expectedIPs) {
      await expect(dialog.getByText(ip, { exact: true })).toBeVisible({ timeout: 1_000 });
    }
  }).toPass({ timeout: remainingBudget(), intervals: [2_000] });
}

test.describe('first run over a NIAC link', () => {
  test.setTimeout(ACCEPTANCE_MS + 30_000);

  test('Network and Security list every scenario device within 90 s of start', async ({
    page,
  }) => {
    expect(Number.isFinite(startedAt)).toBe(true);
    expect(expectedIPs.length).toBeGreaterThan(0);

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
