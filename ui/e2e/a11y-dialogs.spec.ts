/**
 * a11y-dialogs.spec.ts — the dialogs an operator opens from a card (#388).
 *
 * `a11y-routes.spec.ts` walks the pages and the shell's three overlays; the
 * dialogs below open only from a click on a card or a menu, and the Bluetooth
 * table only once a scan has found something, so nothing reached them in the
 * assembled app. Each gets axe, a Tab trap check, and the check axe cannot
 * make: Escape closes it and hands focus back to a control outside it, not to
 * the top of the page.
 */
import { expect, type Locator, type Page, test } from '@playwright/test';
import axe from 'axe-core';

import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';
import { axeViolations } from './helpers/axe';
import { mockBluetoothScanJob } from './helpers/bluetooth';
import { expectFocused } from './helpers/focus';

type Dialog = {
  name: string;
  route: string;
  /** Brings the page to where the dialog can open; returns the opening control. */
  prepare: (page: Page) => Promise<Locator>;
  /** Where focus belongs once the dialog closes. Defaults to the opener. */
  returnTo?: (page: Page) => Locator;
  /** The dialog's own `aria-labelledby` id, or a test id inside it. */
  dialog: (page: Page) => Locator;
};

const byLabelledBy = (id: string) => (page: Page) =>
  page.locator(`[role="dialog"][aria-labelledby="${id}"]`);

const DIALOGS: Dialog[] = [
  {
    name: 'Bluetooth device table',
    route: '/security',
    prepare: async (page) => {
      await mockBluetoothScanJob(page);
      await page.getByTestId('bluetooth-scan-button').click();
      const open = page.getByTestId('bluetooth-card-maximize');
      await expect(open).toBeEnabled();
      return open;
    },
    dialog: (page) =>
      page.locator('[role="dialog"]').filter({ has: page.getByTestId('bluetooth-modal') }),
  },
  {
    name: 'discovery table',
    route: '/network',
    prepare: async (page) => page.getByTestId('discovery-card-maximize'),
    dialog: byLabelledBy('discovery-modal-title'),
  },
  {
    name: 'log viewer',
    route: '/logs',
    prepare: async (page) => page.getByTestId('logs-card-maximize'),
    dialog: byLabelledBy('log-viewer-modal-title'),
  },
  {
    // "Manage profiles" lives in the account menu, which closes as the dialog
    // opens, so the control focus can return to is the account button.
    name: 'profile manager',
    route: '/link',
    prepare: async (page) => {
      await page.getByTestId('rail-account').click();
      return page.getByTestId('rail-profile-manage');
    },
    returnTo: (page) => page.getByTestId('rail-account'),
    dialog: byLabelledBy('profile-modal-title'),
  },
];

/**
 * A focused control's tooltip takes the first Escape (WCAG 1.4.13: content
 * shown on focus is dismissable without moving focus), so the dialog may need
 * a second one. Both steps are asserted, not just the end state.
 */
async function closeWithEscape(page: Page, dialog: Locator): Promise<void> {
  const tooltip = page.getByRole('tooltip');
  if ((await tooltip.count()) > 0) {
    await page.keyboard.press('Escape');
    await expect(tooltip).toHaveCount(0);
    await expect(dialog).toBeVisible();
  }
  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
}

test.use({ storageState: AUTH_STORAGE_STATE });

test.beforeEach(async ({ page }) => {
  await disableAnimations(page);
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.addInitScript({ content: axe.source });
  await page.setViewportSize({ width: 1440, height: 900 });
});

for (const { name, route, prepare, returnTo, dialog } of DIALOGS) {
  test(`the ${name} passes axe, traps Tab and restores focus on Escape`, async ({ page }) => {
    await page.goto(route);
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    const opener = await prepare(page);
    await opener.focus();
    await page.keyboard.press('Enter');
    await expect(dialog(page)).toBeVisible();
    await page.waitForLoadState('networkidle');

    expect(await axeViolations(page)).toEqual([]);

    for (const key of ['Tab', 'Shift+Tab']) {
      for (let presses = 0; presses < 30; presses++) {
        await page.keyboard.press(key);
        expect(
          await dialog(page).evaluate((root) => root.contains(document.activeElement)),
          `${key} #${presses + 1} left the ${name}`,
        ).toBe(true);
      }
    }

    await closeWithEscape(page, dialog(page));
    await expectFocused((returnTo ?? (() => opener))(page));
  });
}

test('the profile editor passes axe and returns focus to the profile manager', async ({ page }) => {
  await page.goto('/link');
  await expect(page.getByTestId('page-header-title')).toBeVisible();
  await page.getByTestId('rail-account').click();
  await page.getByTestId('rail-profile-manage').click();
  const create = page.getByTestId('profile-create');
  await create.focus();
  await page.keyboard.press('Enter');
  const editor = byLabelledBy('profile-editor-title')(page);
  await expect(editor).toBeVisible();

  expect(await axeViolations(page, '[aria-labelledby="profile-editor-title"]')).toEqual([]);
  for (let presses = 0; presses < 20; presses++) {
    await page.keyboard.press('Tab');
    expect(
      await editor.evaluate((root) => root.contains(document.activeElement)),
      `Tab #${presses + 1} left the profile editor`,
    ).toBe(true);
  }

  // Escape closes the editor only; the manager it was opened from stays.
  await closeWithEscape(page, editor);
  await expect(byLabelledBy('profile-modal-title')(page)).toBeVisible();
  await expectFocused(create);
});

// Nothing opened the vulnerability details until the CVE badge became its
// opener (#2640), so this is the first time a browser has reached it. The
// devices and the findings are mocked: a test host has no CVE data.
test('the vulnerability details pass axe and return focus to the CVE badge', async ({ page }) => {
  const findings = {
    deviceIp: '10.44.10.7',
    hostname: 'sw-core-01',
    vendor: 'Cisco',
    product: 'IOS',
    version: '15.2',
    scanTime: new Date().toISOString(),
    vulnerabilities: [
      {
        cveId: 'CVE-2026-0002',
        description: 'Remote code execution in the web UI.',
        severity: 'CRITICAL',
        score: 9.8,
        published: '2026-01-01T00:00:00Z',
        modified: '2026-01-01T00:00:00Z',
        references: ['https://nvd.nist.gov/vuln/detail/CVE-2026-0002'],
        affectedCpe: '',
      },
    ],
  };
  const devices = [
    {
      ip: '10.44.10.7',
      mac: '00:1b:21:aa:bb:01',
      hostname: 'sw-core-01',
      discoveryMethod: ['arp'],
      lastSeen: new Date().toISOString(),
      isLocal: true,
      vulnerabilities: findings,
    },
  ];
  await page.route('**/api/v1/security/devices', (route) => route.fulfill({ json: { devices } }));
  await page.route('**/api/v1/security/vulnerabilities/device?*', (route) =>
    route.fulfill({ json: findings }),
  );

  await page.goto('/network');
  await expect(page.getByTestId('page-header-title')).toBeVisible();
  await page.getByTestId('discovery-card-maximize').focus();
  await page.keyboard.press('Enter');
  const table = byLabelledBy('discovery-modal-title')(page);
  await expect(table).toBeVisible();

  const badge = table.getByRole('button', { name: 'Show vulnerabilities for sw-core-01' });
  await badge.focus();
  await page.keyboard.press('Enter');
  const details = page.getByRole('dialog', { name: 'Vulnerability Report' });
  await expect(details.getByText('CVE-2026-0002')).toBeVisible();

  expect(await axeViolations(page, '[aria-labelledby="modal-title"]')).toEqual([]);
  for (let presses = 0; presses < 20; presses++) {
    await page.keyboard.press('Tab');
    expect(
      await details.evaluate((root) => root.contains(document.activeElement)),
      `Tab #${presses + 1} left the vulnerability details`,
    ).toBe(true);
  }

  // Escape closes the details only; the discovery table stays open.
  await closeWithEscape(page, details);
  await expect(table).toBeVisible();
  await expectFocused(badge);
});
