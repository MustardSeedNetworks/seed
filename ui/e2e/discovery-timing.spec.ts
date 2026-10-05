import { expect, type Page, test } from '@playwright/test';
import { sidebarSettingsButton, skipSetupWizard } from './helpers/auth';

/**
 * Discovery timing in the settings drawer (seed#491): the two timers the
 * daemon reads, edited in minutes and seconds and stored in milliseconds.
 * Arrow-key stepping is a browser behaviour jsdom does not implement, so it is
 * proved here rather than in the component test.
 */

const SETTINGS_PATH = '/api/v1/security/devices/settings';

async function openDiscoverySettings(page: Page): Promise<void> {
  await page.goto('/');
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
  await sidebarSettingsButton(page).click();
  await expect(page.getByTestId('settings-drawer')).toBeVisible();
  await page.getByTestId('discovery-settings-section').getByRole('button').first().click();
  await expect(page.getByTestId('discovery-rescan-interval')).toBeVisible();
}

function savedSettings(page: Page): Promise<Record<string, unknown>> {
  return page
    .waitForRequest((req) => req.method() === 'PUT' && req.url().endsWith(SETTINGS_PATH))
    .then((req) => req.postDataJSON() as Record<string, unknown>);
}

async function storedRescanMinutes(page: Page): Promise<number> {
  const res = await page.request.get(SETTINGS_PATH);
  const body = (await res.json()) as { timing: { rescanIntervalMs: number } };
  expect(Object.keys(body.timing)).toEqual(['rescanIntervalMs']);
  return body.timing.rescanIntervalMs / 60000;
}

test.describe('Discovery timing settings', () => {
  // Every test rewrites the daemon's one discovery config.
  test.describe.configure({ mode: 'serial' });

  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('shows two timers in minutes and seconds and saves them in milliseconds', async ({
    page,
  }) => {
    await openDiscoverySettings(page);

    const section = page.getByTestId('discovery-settings-section');
    for (const gone of [/ping timeout/i, /probe interval/i, /scan workers/i, /banner timeout/i]) {
      await expect(section.getByText(gone)).toHaveCount(0);
    }

    const rescan = page.getByTestId('discovery-rescan-interval');
    // The field shows a placeholder until the drawer's load lands, so wait for
    // the stored value before choosing one the daemon does not hold.
    const held = await storedRescanMinutes(page);
    await expect(rescan).toHaveValue(String(held));
    const minutes = held === 3 ? 4 : 3;
    const afterRescan = savedSettings(page);
    await rescan.fill(String(minutes));
    expect((await afterRescan).timing).toEqual({ rescanIntervalMs: minutes * 60000 });

    await page.getByTestId('discovery-timing-advanced').getByRole('button').click();
    const limit = page.getByTestId('discovery-scan-timeout');
    await limit.focus();
    const afterLimit = savedSettings(page);
    await page.keyboard.press('ArrowUp');
    const body = await afterLimit;
    expect(body.scanTimeoutMs).toBe(Number(await limit.inputValue()) * 1000);
    expect(body).not.toHaveProperty('arpScanWorkers');
    expect(body).not.toHaveProperty('pingTimeoutMs');
    expect(body).not.toHaveProperty('scanIntervalMs');

    await page.reload();
    await openDiscoverySettings(page);
    await expect(page.getByTestId('discovery-rescan-interval')).toHaveValue(String(minutes));
    expect(await storedRescanMinutes(page)).toBe(minutes);
  });

  test('refuses an out-of-range value and restores the stored one on blur', async ({ page }) => {
    await openDiscoverySettings(page);

    const rescan = page.getByTestId('discovery-rescan-interval');
    const before = await rescan.inputValue();
    let wrote = false;
    page.on('request', (req) => {
      if (req.method() === 'PUT' && req.url().endsWith(SETTINGS_PATH)) {
        wrote = true;
      }
    });
    await rescan.fill('0');
    await rescan.blur();
    await expect(rescan).toHaveValue(before);
    expect(wrote).toBe(false);
  });
});
