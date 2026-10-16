import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Device configs (UI-SEED-25): on a licence with configuration backup, the
 * newest saved backup of a device opens with its change highlighted, and a
 * failed attempt says why it saved nothing.
 *
 * The suite's daemon is unlicensed and has no SSH device to back up, so the
 * licence and the three backup reads are answered with the shapes
 * internal/deviceconfig serializes. The unlicensed page is covered by
 * license-feature-gate.spec.ts.
 */

async function fulfil(page: Page, path: string, body: unknown): Promise<void> {
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

const TARGET = 'dct-0000000000e2';
const SAVED = 'dcb-0000000000b2';
const FAILED = 'dcb-0000000000b3';

test('an operator opens a device and sees what changed in its configuration', async ({ page }) => {
  await fulfil(page, '/api/v1/license', {
    tier: 'Pro',
    tierValue: 2,
    isTrialMode: false,
    canMintTokens: true,
    activated: true,
    features: ['compliance_advanced'],
  });
  await fulfil(page, '/api/v1/device-config-targets', {
    count: 1,
    targets: [
      {
        id: TARGET,
        name: 'access-sw-07',
        host: '192.0.2.7',
        port: 22,
        platform: 'cisco-ios',
        updatedAt: '2026-10-15T09:00:00Z',
      },
    ],
  });
  await fulfil(page, '/api/v1/device-configs', {
    count: 3,
    backups: [
      {
        id: FAILED,
        targetId: TARGET,
        takenAt: '2026-10-15T12:00:00Z',
        status: 'auth_failed',
        error: 'ssh: unable to authenticate',
      },
      { id: SAVED, targetId: TARGET, takenAt: '2026-10-15T11:00:00Z', status: 'ok', sha256: 'b' },
      {
        id: 'dcb-0000000000b1',
        targetId: TARGET,
        takenAt: '2026-10-15T10:00:00Z',
        status: 'ok',
        sha256: 'a',
      },
    ],
  });
  await fulfil(page, `/api/v1/device-configs/${SAVED}/diff`, {
    targetId: TARGET,
    backupId: SAVED,
    takenAt: '2026-10-15T11:00:00Z',
    previousBackupId: 'dcb-0000000000b1',
    previousTakenAt: '2026-10-15T10:00:00Z',
    added: 1,
    removed: 1,
    unified:
      '--- dcb-0000000000b1\n+++ dcb-0000000000b2\n@@ -4,3 +4,3 @@\n interface Vlan20\n- ip helper-address 10.0.0.5\n+ ip helper-address 10.0.0.6\n !\n',
  });

  await skipSetupWizard(page);
  await page.goto('/device-configs');

  const diff = page.getByTestId('config-diff');
  await expect(diff).toBeVisible({ timeout: 10000 });
  await expect(diff.locator('[data-kind="removed"]')).toHaveText('- ip helper-address 10.0.0.5');
  await expect(diff.locator('[data-kind="added"]')).toHaveText('+ ip helper-address 10.0.0.6');
  await expect(page.getByTestId('config-diff-counts')).toContainText('+1 −1 lines');
  await expect(page.getByTestId(`config-backup-${SAVED}`)).toHaveAttribute('aria-pressed', 'true');

  await page.getByTestId(`config-backup-${FAILED}`).click();
  await expect(page.getByTestId('config-backup-failed')).toContainText('Sign-in failed');
  await expect(diff).toBeHidden();
});
