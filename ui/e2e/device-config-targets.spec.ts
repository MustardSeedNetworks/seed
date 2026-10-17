import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Configuration backup management (UI-SEED-45): an operator adds a device
 * with a new SSH login, backs it up, trusts the host key the first backup
 * recorded, and removes the device.
 *
 * The suite's daemon is unlicensed and has no SSH device to back up, so the
 * licence, the vault, the target routes, the run and its job event are
 * answered with the shapes internal/deviceconfig serializes, statefully, so a
 * write is what the next read reports.
 */

const TARGET = 'dct-0000000000e5';
const JOB = 'job-0000000000e5';
const FINGERPRINT = 'SHA256:q1w2e3r4t5y6u7i8o9p0';

async function fulfil(page: Page, path: string, body: unknown, status = 200): Promise<void> {
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    },
  );
}

test('an operator adds a device, backs it up, trusts its host key and removes it', async ({
  page,
}) => {
  await fulfil(page, '/api/v1/license', {
    tier: 'Pro',
    tierValue: 2,
    isTrialMode: false,
    canMintTokens: true,
    activated: true,
    features: ['compliance_advanced'],
  });

  const writes: string[] = [];
  let target: Record<string, unknown> | null = null;
  let ran = false;

  await page.route(
    (url) => url.pathname === '/api/v1/device-credentials',
    async (route) => {
      if (route.request().method() === 'POST') {
        writes.push(`POST credential ${JSON.stringify(route.request().postDataJSON())}`);
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ id: 'cred-e5', name: 'lab backup', kind: 'ssh' }),
        });
        return;
      }
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ count: 0, credentials: [] }),
      });
    },
  );
  await page.route(
    (url) => url.pathname.startsWith('/api/v1/device-config-targets'),
    async (route) => {
      const method = route.request().method();
      const path = new URL(route.request().url()).pathname;
      if (method !== 'GET') {
        writes.push(`${method} ${path}`);
      }
      if (method === 'POST') {
        target = {
          ...(route.request().postDataJSON() as Record<string, unknown>),
          id: TARGET,
          port: 22,
          updatedAt: '2026-10-17T07:00:00Z',
        };
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify(target),
        });
        return;
      }
      if (method === 'PUT' && target) {
        target = { ...target, hostKeySha256: FINGERPRINT, pendingHostKeySha256: undefined };
      } else if (method === 'DELETE') {
        target = null;
      }
      if (method !== 'GET') {
        await route.fulfill({ status: 204 });
        return;
      }
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ count: target ? 1 : 0, targets: target ? [target] : [] }),
      });
    },
  );
  await page.route(
    (url) => url.pathname === '/api/v1/device-configs/run',
    async (route) => {
      writes.push(`POST run ${JSON.stringify(route.request().postDataJSON())}`);
      ran = true;
      // The first backup meets an unpinned key: it records it and saves nothing.
      if (target) {
        target = { ...target, pendingHostKeySha256: FINGERPRINT };
      }
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: JOB,
          kind: 'device-config-backup',
          state: 'queued',
          progress: 0,
        }),
      });
    },
  );
  await page.route(
    (url) => url.pathname === '/api/v1/jobs/events',
    async (route) => {
      const frame = ran
        ? `event: job\ndata: ${JSON.stringify({
            id: JOB,
            kind: 'device-config-backup',
            state: 'succeeded',
            progress: 1,
            result: { results: [], succeeded: 0, failed: 1 },
          })}\n\n`
        : '';
      await route.fulfill({
        contentType: 'text/event-stream',
        body: `retry: 250\n\n${frame}`,
      });
    },
  );
  await page.route(
    (url) => url.pathname === '/api/v1/device-configs',
    async (route) => {
      const backups = ran
        ? [
            {
              id: 'dcb-0000000000e5',
              targetId: TARGET,
              takenAt: '2026-10-17T07:01:00Z',
              status: 'host_key_unconfirmed',
              error: 'host key not confirmed',
            },
          ]
        : [];
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ count: backups.length, backups }),
      });
    },
  );

  await skipSetupWizard(page);
  await page.goto('/device-configs');

  await page.getByTestId('config-target-add').click({ timeout: 10000 });
  await page.getByTestId('config-target-name').fill('edge-05');
  await page.getByTestId('config-target-host').fill('192.0.2.5');
  await page.getByTestId('config-target-platform').selectOption('arista-eos');
  await expect(page.getByTestId('config-target-new-login')).toBeVisible();
  await page.getByTestId('config-target-login-name').fill('lab backup');
  await page.getByTestId('config-target-login-user').fill('backup');
  await page.getByTestId('config-target-login-password').fill('hunter2');
  await page.getByTestId('config-target-save').click();

  await expect(page.getByTestId(`config-target-${TARGET}`)).toContainText('edge-05');
  await expect(page.getByTestId('config-host-key-none')).toBeVisible();

  await page.getByTestId('config-run-one').click();
  await expect(page.getByTestId('config-run-status')).toHaveText(
    'Backup finished: 0 saved, 1 failed.',
  );
  await expect(page.getByTestId('config-host-key-pending')).toHaveText(FINGERPRINT);
  await expect(page.getByTestId('config-backup-failed')).toContainText('Host key not confirmed');

  await page.getByTestId('config-host-key-confirm').click();
  await expect(page.getByTestId('config-host-key-pinned')).toHaveText(FINGERPRINT);

  await page.getByTestId('config-target-remove').click();
  await page.getByTestId('confirm-modal-confirm').click();
  await expect(page.getByText('No devices are backed up yet.')).toBeVisible();

  expect(writes).toEqual([
    'POST credential {"name":"lab backup","sshUser":"backup","sshPassword":"hunter2"}',
    'POST /api/v1/device-config-targets',
    `POST run {"targetIds":["${TARGET}"]}`,
    `PUT /api/v1/device-config-targets/${TARGET}/host-key`,
    `DELETE /api/v1/device-config-targets/${TARGET}`,
  ]);
});
