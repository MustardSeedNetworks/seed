import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Card data tables must fit their card (#2708).
 *
 * `NeighbourCacheCard` and `BonjourCard` wrap their tables in
 * `overflow-x-auto`, so a table wider than the card scrolls sideways inside it
 * and the last column sits off-card. The page-level gate in
 * minimum-width.spec.ts cannot see this: the card clips its own content, so the
 * document never overflows. This measures the scroll container itself.
 *
 * The rows are mocked rather than taken from the daemon's real ARP table and
 * mDNS browse, for two reasons: a test host has no deterministic neighbours, so
 * the tables would often be empty and the assertion would pass without ever
 * rendering a table; and the widths that matter are the worst realistic ones --
 * a full IPv6 address, a 17-character MAC and a long OUI vendor string.
 */

const NEIGHBOURS = {
  total: 4,
  entries: [
    {
      ip: '192.168.1.1',
      mac: 'a4:2b:8c:1d:5e:9f',
      vendor: 'Cisco Systems, Inc.',
      interface: 'en0',
      state: 'reachable',
      family: 'ipv4',
    },
    {
      // The widest realistic row: a full IPv6 address beside the longest
      // vendor strings the IEEE OUI registry actually contains.
      ip: '2001:0db8:85a3:0000:0000:8a2e:0370:7334',
      mac: 'f0:9f:c2:3a:7b:11',
      vendor: 'Hewlett Packard Enterprise Networking',
      interface: 'en0',
      state: 'stale',
      family: 'ipv6',
    },
    {
      ip: '192.168.1.42',
      mac: 'b8:27:eb:44:2c:d0',
      vendor: 'Raspberry Pi Foundation',
      interface: 'bridge100',
      state: 'reachable',
      family: 'ipv4',
    },
    {
      ip: '192.168.1.77',
      mac: '3c:22:fb:90:1a:6e',
      vendor: 'Apple, Inc.',
      interface: 'en0',
      state: 'delay',
      family: 'ipv4',
    },
  ],
};

const BONJOUR = {
  serviceTypes: ['_http._tcp', '_ipp._tcp'],
  reflector: { state: 'off' },
  responsesObserved: 3,
  truncated: false,
  durationMs: 120,
  services: [
    {
      instance: 'Office Colour LaserJet MFP M480f',
      type: '_ipp._tcp.local.',
      host: 'office-laserjet-m480f.local.',
      port: 631,
      addresses: ['192.168.1.30'],
      origin: 'local',
    },
    {
      instance: 'Conference Room Display',
      type: '_airplay._tcp.local.',
      host: 'conference-room-display.local.',
      port: 7000,
      addresses: ['2001:0db8:85a3:0000:0000:8a2e:0370:7334'],
      origin: 'local',
    },
  ],
};

/**
 * Every in-page card table, with how far it exceeds the box that holds it.
 *
 * Found via the table rather than via `.overflow-x-auto`: the fix removes that
 * class, and a selector naming it would report zero tables and pass vacuously
 * on exactly the change it is meant to verify.
 */
async function cardTableOverflow(
  page: import('@playwright/test').Page,
): Promise<{ label: string; overflow: number }[]> {
  return page.evaluate(() => {
    const out: { label: string; overflow: number }[] = [];
    for (const table of Array.from(document.querySelectorAll('table'))) {
      // A modal is allowed to scroll (the Logs viewer); only in-page cards are
      // under this rule, and a closed modal is not in the DOM to begin with.
      if (table.closest('[role="dialog"]')) {
        continue;
      }
      const box = table.parentElement;
      if (!box) {
        continue;
      }
      const card = table.closest('[data-testid], section, article');
      const heading = card?.querySelector('h2, h3');
      out.push({
        label: heading?.textContent?.trim() ?? card?.getAttribute('data-testid') ?? 'unknown',
        // Both readings matter: the box scrolling means content is hidden
        // behind a scrollbar, and the table being wider than the box means it
        // is clipped even where nothing scrolls.
        overflow: Math.max(box.scrollWidth - box.clientWidth, table.scrollWidth - box.clientWidth),
      });
    }
    return out;
  });
}

for (const width of [1280, 1440]) {
  test(`card data tables fit their card at ${width}px`, async ({ page }) => {
    await page.route('**/api/v1/network/neighbours', (route) =>
      route.fulfill({ json: NEIGHBOURS }),
    );
    await page.route('**/api/v1/discovery/bonjour', (route) => route.fulfill({ json: BONJOUR }));

    await page.setViewportSize({ width, height: 900 });
    await skipSetupWizard(page);
    await page.goto('/network');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

    // The Bonjour browse is deliberately not run on mount -- it sends
    // multicast and listens for seconds -- so its table only exists after the
    // operator asks for it.
    await page.getByTestId('bonjour-browse').click();

    // Both tables must actually be on the page before anything is measured,
    // or the assertion passes against nothing. That guard already earned its
    // place: the first run of this spec found one table, not two, because the
    // Bonjour browse above was missing. They are waited for individually
    // rather than by a total count, so adding an unrelated table to this page
    // does not fail the spec for the wrong reason.
    await expect(page.getByTestId('bonjour-services').locator('tr')).not.toHaveCount(0, {
      timeout: 10000,
    });
    await expect(page.locator('table').filter({ hasText: NEIGHBOURS.entries[0].mac })).toHaveCount(
      1,
      { timeout: 10000 },
    );

    const tables = await cardTableOverflow(page);
    expect(
      tables.length,
      'expected at least the two card tables to be measured',
    ).toBeGreaterThanOrEqual(2);

    const cut = tables.filter((t) => t.overflow > 0);
    expect(cut, `these card tables are cut off at ${width}px: ${JSON.stringify(cut)}`).toEqual([]);
  });
}
