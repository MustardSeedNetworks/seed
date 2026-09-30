import { expect, type Page, test } from '@playwright/test';
import { AUTH_STORAGE_STATE, disableAnimations } from './helpers/auth';

/**
 * The discovery table fits the modal at every published width and stays
 * usable with a long device list (#461).
 *
 * The table used to be `min-w-[900px]`, so below that it scrolled sideways
 * inside the modal, and it mounted every row: opening it on 1,000 devices
 * blocked the main thread for 777 ms, on 5,000 for 3.8 s (Chromium,
 * dev-srv-ubuntu). Lower-priority columns now fold into each row's details
 * panel as the modal narrows, and a list above the threshold mounts only the
 * rows in view.
 *
 * Devices are mocked: a test host discovers an unpredictable handful, and the
 * widths that matter are the worst realistic ones (a full IPv6 address, a long
 * OUI vendor, a long hostname).
 */

test.use({ storageState: AUTH_STORAGE_STATE });

const LONG_VENDOR = 'Hewlett Packard Enterprise Networking';

function device(i: number) {
  return {
    ip: `10.${Math.floor(i / 250)}.${i % 250}.${(i % 200) + 1}`,
    ipv6: i === 0 ? '2001:0db8:85a3:0000:0000:8a2e:0370:7334' : undefined,
    mac: `f0:9f:c2:${(Math.floor(i / 256) % 256).toString(16).padStart(2, '0')}:${(i % 256).toString(16).padStart(2, '0')}:11`,
    vendor: LONG_VENDOR,
    hostname: `access-switch-${i}.north-campus.corp.example.internal`,
    discoveryMethod: ['arp', 'snmp', 'lldp'],
    lastSeen: new Date().toISOString(),
    firstSeen: new Date().toISOString(),
    isLocal: true,
  };
}

function mountedRows(page: Page) {
  return page.getByTestId('discovery-table').locator('tbody[data-index]');
}

async function openWithDevices(page: Page, count: number, width: number): Promise<void> {
  const devices = Array.from({ length: count }, (_, i) => device(i));
  await page.route('**/api/v1/security/devices', (route) => route.fulfill({ json: { devices } }));
  await page.route('**/api/v1/security/devices/status', (route) =>
    route.fulfill({
      json: {
        scanning: false,
        deviceCount: count,
        lastScan: new Date().toISOString(),
        subnet: '10.0.0.0/16',
        localIP: '10.0.0.2',
        interface: 'eth0',
      },
    }),
  );
  await disableAnimations(page);
  await page.setViewportSize({ width, height: 900 });
  await page.goto('/network');
  // From the keyboard, not a click: the cards above this one load real daemon
  // data while Playwright scrolls down to it, and a click at the button's
  // position a moment earlier opened nothing on a loaded WebKit (#2922).
  await page.getByTestId('discovery-card-maximize').focus();
  await page.keyboard.press('Enter');
  // The rows must be on screen before anything is measured, or every
  // assertion below passes against an empty table.
  await expect(mountedRows(page).first()).toBeVisible({ timeout: 20000 });
}

for (const width of [480, 768, 1280, 1440]) {
  test(`the discovery table fits the modal at ${width}px`, async ({ page }) => {
    await openWithDevices(page, 12, width);

    const overflow = await page.getByTestId('discovery-table').evaluate((table) => {
      const box = table.parentElement;
      if (!box) {
        throw new Error('the table has no scroll box');
      }
      return Math.max(box.scrollWidth - box.clientWidth, table.scrollWidth - box.clientWidth);
    });
    expect(overflow, `the table scrolls sideways by ${overflow}px`).toBeLessThanOrEqual(0);

    const dialogOverflow = await page
      .getByRole('dialog')
      .evaluate((dialog) => dialog.scrollWidth - dialog.clientWidth);
    expect(dialogOverflow, `the modal scrolls sideways by ${dialogOverflow}px`).toBeLessThanOrEqual(
      0,
    );
  });
}

test('a column folded away at the minimum width is in the details panel', async ({ page }) => {
  await openWithDevices(page, 12, 480);
  const dialog = page.getByRole('dialog');
  await expect(dialog.getByRole('columnheader', { name: 'MAC' })).toBeHidden();
  await expect(dialog.getByRole('columnheader', { name: 'Vendor' })).toBeHidden();
  await expect(dialog.getByText(device(0).mac)).toBeHidden();

  const toggle = page.getByTestId('discovery-row-toggle').first();
  await toggle.click();
  await expect(toggle).toHaveAttribute('aria-expanded', 'true');
  const summary = page.getByTestId('discovery-row-summary').first();
  await expect(summary.getByText(device(0).mac)).toBeVisible();
  await expect(summary.getByText(LONG_VENDOR)).toBeVisible();
});

test('every column is a column at full width, with nothing duplicated', async ({ page }) => {
  await openWithDevices(page, 12, 1440);
  const dialog = page.getByRole('dialog');
  for (const name of ['MAC', 'Vendor', 'Discovery', 'Ports', 'CVEs', 'Last Seen']) {
    await expect(dialog.getByRole('columnheader', { name })).toBeVisible();
  }
  // These rows carry no protocol, SNMP or port detail, so at full width they
  // have nothing to expand.
  await expect(page.getByTestId('discovery-row-toggle').first()).toBeHidden();
});

test('sorting and expanding work from the keyboard', async ({ page }) => {
  await openWithDevices(page, 12, 768);
  const header = page.getByRole('columnheader', { name: 'Hostname' });
  await expect(header).toHaveAttribute('aria-sort', 'none');
  await header.getByRole('button').focus();
  await page.keyboard.press('Enter');
  await expect(header).toHaveAttribute('aria-sort', 'ascending');
  await page.keyboard.press('Enter');
  await expect(header).toHaveAttribute('aria-sort', 'descending');

  const toggle = page.getByTestId('discovery-row-toggle').first();
  await toggle.focus();
  await page.keyboard.press('Enter');
  await expect(toggle).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByTestId('discovery-row-summary').first()).toBeVisible();
  await page.keyboard.press('Space');
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
});

test('a short list mounts every row', async ({ page }) => {
  await openWithDevices(page, 150, 1440);
  await expect(mountedRows(page)).toHaveCount(150);
});

test('a long list mounts only the rows in view, and Tab walks past them', async ({ page }) => {
  await openWithDevices(page, 5000, 1440);
  const initial = await mountedRows(page).count();
  expect(initial, 'a 5,000-device list mounted too many rows').toBeLessThan(60);
  const lastMounted = Number(await mountedRows(page).last().getAttribute('data-index'));

  // At full width a row's only control is its Scan button (these rows have
  // no details to toggle). Tabbing well past the rows mounted at first has to
  // reach rows that were not in the DOM when it started.
  const firstScan = mountedRows(page).first().getByRole('button', { name: 'Scan' });
  await firstScan.focus();
  await expect(firstScan).toBeFocused();
  for (let i = 0; i < lastMounted + 10; i++) {
    await page.keyboard.press('Tab');
  }
  // Polled: at the edge of the mounted rows focus moves once the next row is
  // mounted, and on a loaded WebKit the last Tab's move had not landed when a
  // single read looked (#2922).
  await expect
    .poll(() => focusedRowIndex(page), {
      message: 'focus left the table instead of reaching later rows',
    })
    .toBeGreaterThan(lastMounted);
  await expect(page.locator(':focus')).toBeInViewport();
  expect(await mountedRows(page).count()).toBeLessThan(60);
});

function focusedRowIndex(page: Page): Promise<number> {
  return page.evaluate(() =>
    Number(document.activeElement?.closest('tbody')?.getAttribute('data-index') ?? -1),
  );
}

// The deterministic form of the walk above. Focusing without scrolling leaves
// the next row unmounted, which is the moment a fast Tab used to hit: native
// order had nowhere to go inside the table, so the dialog's trap wrapped focus
// to the Rescan button (seen on WebKit under load).
test('Tab and Shift+Tab cross the edge of the mounted rows', async ({ page }) => {
  await openWithDevices(page, 5000, 1440);
  const scanIn = (row: ReturnType<typeof mountedRows>) =>
    row.getByRole('button', { name: 'Scan' }).evaluate((b) => b.focus({ preventScroll: true }));

  const lastIndex = Number(await mountedRows(page).last().getAttribute('data-index'));
  await scanIn(mountedRows(page).last());
  // Every stop focus makes on the way, not only where it ends: a detour
  // through the Rescan button is announced by a screen reader even when focus
  // then lands on the right row.
  await page.evaluate(() => {
    const stops: string[] = [];
    (window as unknown as { __focusStops: string[] }).__focusStops = stops;
    document.addEventListener('focusin', (event) => {
      const target = event.target as HTMLElement;
      stops.push(target.closest('tbody[data-index]') ? 'row' : (target.textContent ?? '?'));
    });
  });
  await page.keyboard.press('Tab');
  await expect.poll(() => focusedRowIndex(page)).toBe(lastIndex + 1);
  await expect(page.locator(':focus')).toBeInViewport();
  const stops = await page.evaluate(
    () => (window as unknown as { __focusStops: string[] }).__focusStops,
  );
  expect(
    stops.filter((stop) => stop !== 'row'),
    'focus stopped outside the rows',
  ).toEqual([]);

  await page.getByTestId('discovery-table').evaluate((table) => {
    if (table.parentElement) {
      table.parentElement.scrollTop = 30000;
    }
  });
  await expect
    .poll(async () => Number(await mountedRows(page).first().getAttribute('data-index')))
    // 30,000 px is some 490 rows down; anything short of that is mid-update.
    .toBeGreaterThan(400);
  const firstIndex = Number(await mountedRows(page).first().getAttribute('data-index'));
  await scanIn(mountedRows(page).first());
  await page.keyboard.press('Shift+Tab');
  await expect.poll(() => focusedRowIndex(page)).toBe(firstIndex - 1);
  await expect(page.locator(':focus')).toBeInViewport();
});
