import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * TFTP session card (UI-SEED-44): an admin starts the TFTP server from the
 * device configs page with uploads off, sees where it listens, and stops it.
 *
 * The suite's daemon is unlicensed and cannot bind UDP port 69, so the
 * licence and the session route are answered with the shapes internal/tftp
 * serializes, statefully, so a start is what the next read reports.
 */

const RUNNING = {
  running: true,
  interface: 'eth0',
  address: '192.0.2.10',
  port: 69,
  directory: '/var/lib/seed/tftp',
  uploadDirectory: '/var/lib/seed/tftp-incoming',
  allowUpload: false,
  startedAt: '2026-10-17T06:00:00Z',
  expiresAt: '2026-10-17T07:00:00Z',
  idleTimeoutSeconds: 600,
  activeTransfers: 0,
};

async function json(page: Page, path: string, body: unknown): Promise<void> {
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(body),
      });
    },
  );
}

test('an admin starts and stops a TFTP session with uploads off', async ({ page }) => {
  await json(page, '/api/v1/license', {
    tier: 'Pro',
    tierValue: 2,
    isTrialMode: false,
    canMintTokens: true,
    activated: true,
    features: ['compliance_advanced'],
  });
  await json(page, '/api/v1/device-config-targets', { count: 0, targets: [] });

  const starts: unknown[] = [];
  let session: Record<string, unknown> = { running: false, allowUpload: false, activeTransfers: 0 };
  await page.route(
    (url) => url.pathname === '/api/v1/tftp/session',
    async (route) => {
      const method = route.request().method();
      if (method === 'POST') {
        const body = route.request().postDataJSON() as { interface: string; allowUpload: boolean };
        starts.push(body);
        session = { ...RUNNING, interface: body.interface, allowUpload: body.allowUpload };
      } else if (method === 'DELETE') {
        session = { running: false, allowUpload: false, activeTransfers: 0 };
      }
      await route.fulfill({
        status: method === 'POST' ? 201 : 200,
        contentType: 'application/json',
        body: JSON.stringify(session),
      });
    },
  );

  await skipSetupWizard(page);
  await page.goto('/device-configs');

  const card = page.getByTestId('tftp-session');
  await expect(card).toBeVisible({ timeout: 10000 });
  await expect(page.getByTestId('tftp-allow-upload')).not.toBeChecked();
  await page.getByTestId('tftp-interface').fill('eth0');
  await page.getByTestId('tftp-start').click();

  await expect(page.getByTestId('tftp-listening')).toContainText('eth0 · 192.0.2.10:69');
  await expect(page.getByTestId('tftp-uploads')).toContainText('Off');
  expect(starts).toEqual([{ interface: 'eth0', allowUpload: false }]);

  await page.getByTestId('tftp-stop').click();
  await expect(page.getByTestId('tftp-start')).toBeVisible();
  await expect(page.getByTestId('tftp-listening')).toBeHidden();
});
