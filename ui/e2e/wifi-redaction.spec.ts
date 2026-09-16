import { expect, test } from '@playwright/test';
import type { WiFiResponse } from '../src/types/generated/wifi-response';
import { skipSetupWizard } from './helpers/auth';

const reason =
  'macOS Location Services withheld the network details from Seed. The Wi-Fi connection may still be active.';
const remediation =
  'Check the connection in System Settings > Wi-Fi. If the Seed Wi-Fi helper is installed, allow its Location Services access and keep its user session signed in. The standalone archive cannot request this permission.';

test.beforeEach(async ({ page }) => {
  await skipSetupWizard(page);
  await page.route(/\/api\/v1\/wifi\/wifi\/channel-graph(?:\?|$)/, (route) =>
    route.fulfill({ json: { available: true, error: reason, remediation, data: null } }),
  );
});

for (const path of ['/wifi', '/link']) {
  test(`${path} shows hidden details with actionable guidance`, async ({ page }) => {
    const response: WiFiResponse = {
      interface: 'en0',
      wireless: true,
      status: 'detailsWithheld',
      reason,
      remediation,
    };
    await page.route(/\/api\/v1\/wifi\/wifi(?:\?|$)/, (route) => route.fulfill({ json: response }));
    await page.goto(path);
    const card = page.getByTestId('wifi-details-withheld');
    await expect(card).toBeVisible();
    await expect(card).toContainText('Wi-Fi details hidden');
    await expect(card).toContainText('System Settings > Wi-Fi');
    await expect(card).toContainText('standalone archive cannot request this permission');
    await expect(page.getByTestId('wifi-not-associated')).toHaveCount(0);
    if (path === '/wifi') {
      await expect(page.getByTestId('wifi-scan-error')).toContainText(reason);
      await expect(page.getByTestId('wifi-scan-error')).toContainText('System Settings > Wi-Fi');
      await expect(page.getByText('No networks detected')).toHaveCount(0);
    }
  });
}

test('Wi-Fi page keeps observed association and disconnection distinct', async ({ page }) => {
  let response: WiFiResponse = {
    interface: 'en0',
    wireless: true,
    status: 'associated',
    connected: true,
    ssid: 'permission-test-lab',
    bssid: 'aa:bb:cc:dd:ee:ff',
    channel: 44,
    signal: -52,
    frequency: 5220,
    security: 'WPA2',
  };
  await page.route(/\/api\/v1\/wifi\/wifi(?:\?|$)/, (route) => route.fulfill({ json: response }));
  await page.goto('/wifi');
  await expect(page.getByTestId('wifi-associated')).toContainText('permission-test-lab');
  await expect(page.getByTestId('wifi-associated')).toContainText('-52 dBm');
  await expect(page.getByTestId('wifi-associated')).toContainText('5220 MHz');
  response = { interface: 'en0', wireless: true, status: 'notAssociated', connected: false };
  await page.reload();
  await expect(page.getByTestId('wifi-not-associated')).toContainText('Disconnected');
  await expect(page.getByTestId('wifi-associated')).toHaveCount(0);
});
