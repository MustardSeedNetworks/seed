import { type APIRequestContext, expect, test } from '@playwright/test';

/**
 * Site networks behind the edge router are found from the learned summary
 * alone (seed#2695, seed#2832).
 *
 * Runs only under scripts/e2e-niac-routed.sh, which puts a routed NIAC pack
 * behind a veth pair, gives seed one address on the pack's transit network
 * plus the static route the edge router itself carries for the site, and
 * starts it with nothing but its interface, port and paths configured. The
 * script exports:
 *
 *   SEED_E2E_NIAC_ROUTED     set to 1 to enable this spec
 *   SEED_E2E_SNMP_COMMUNITY  the community the pack's agents answer
 *   SEED_E2E_GATEWAY         the edge router, on the transit network
 *   SEED_E2E_SITE_ROUTE      the host's static route to the site
 *   SEED_E2E_SITE_NETWORKS   comma-separated site CIDRs, inside that route
 *   SEED_E2E_SITE_DEVICES    `cidr=ip|ip;cidr=ip|…`, the devices inside each
 *   SEED_E2E_CORE            the agent whose address table names the site networks
 *
 * The edge router names the site only as its summary route, which is wider
 * than one sweep may probe. With only that summary switched on, the sweep
 * first probes .1 to .4 of each /24 inside it, finds the core switch, and then
 * sweeps the /24s the core's address table names, one per sweep. No site
 * network is typed in and none of the learned /24s is switched on.
 *
 * The 60 s rescan sweeps enabled target networks too (seed#2831), but this
 * spec presses the operator's Scan through the API so each step waits on a
 * sweep it started rather than on the rescan's phase. Only the API is used,
 * because several pages POST a scan of their own on mount (seed#2691) and
 * would put sweeps in the timeline this spec controls.
 */
const RESCAN_MS = 60_000;
// The scan route allows five requests a minute per client.
const SCAN_EVERY_MS = 15_000;
// A learned network needs its device profiled first, so it lands on the scan
// after the one that found the device.
const SCAN_MS = 4 * SCAN_EVERY_MS;
const LEARN_MS = 6 * SCAN_EVERY_MS;
// Probing .1-.4 covers 63 /24s a sweep, so a /16 takes five.
const PROBE_MS = 6 * SCAN_EVERY_MS + SCAN_MS;

const enabled = process.env.SEED_E2E_NIAC_ROUTED === '1';
const community = process.env.SEED_E2E_SNMP_COMMUNITY ?? '';
const gateway = process.env.SEED_E2E_GATEWAY ?? '';
const siteRoute = process.env.SEED_E2E_SITE_ROUTE ?? '';
const core = process.env.SEED_E2E_CORE ?? '';
const siteNetworks = (process.env.SEED_E2E_SITE_NETWORKS ?? '').split(',').filter(Boolean);
const siteDevices = new Map(
  (process.env.SEED_E2E_SITE_DEVICES ?? '')
    .split(';')
    .filter(Boolean)
    .map((entry) => {
      const [cidr = '', ips = ''] = entry.split('=');
      return [cidr, ips.split('|').filter(Boolean)] as const;
    }),
);

test.skip(!enabled, 'needs scripts/e2e-niac-routed.sh (a routed NIAC pack over a veth pair)');

interface Subnet {
  cidr: string;
  name: string;
  enabled: boolean;
  learned: boolean;
}

async function csrfToken(request: APIRequestContext): Promise<string> {
  const response = await request.get('/api/v1/auth/csrf');
  expect(response.status(), 'GET /api/v1/auth/csrf').toBe(200);
  const { token } = (await response.json()) as { token: string };
  return token;
}

async function subnets(request: APIRequestContext): Promise<Subnet[]> {
  const response = await request.get('/api/v1/security/devices/subnets');
  return response.ok() ? ((await response.json()) as Subnet[]) : [];
}

async function discoveredIPs(request: APIRequestContext): Promise<Set<string>> {
  const response = await request.get('/api/v1/security/devices');
  if (!response.ok()) {
    return new Set();
  }
  const { devices } = (await response.json()) as { devices?: { ip?: string }[] };
  return new Set((devices ?? []).map((device) => device.ip ?? ''));
}

/**
 * A poll step that presses Scan when the last press is old enough, so a poll
 * waiting on a sweep's result also drives the sweeps it waits on.
 */
function scanner(request: APIRequestContext): () => Promise<void> {
  let pressedAt = 0;
  return async () => {
    if (Date.now() - pressedAt < SCAN_EVERY_MS) {
      return;
    }
    pressedAt = Date.now();
    const response = await request.post('/api/v1/security/devices/scan', {
      headers: { 'X-CSRF-Token': await csrfToken(request) },
    });
    expect(response.status(), 'POST /api/v1/security/devices/scan').toBe(200);
  };
}

/** The networks that do not yet satisfy `ok`, for a poll to converge on []. */
async function pending(
  networks: string[],
  request: APIRequestContext,
  ok: (subnet: Subnet | undefined) => boolean,
): Promise<string[]> {
  const listed = new Map((await subnets(request)).map((s) => [s.cidr, s]));
  return networks.filter((cidr) => !ok(listed.get(cidr)));
}

test.describe('target networks behind a NIAC edge router', () => {
  test.setTimeout(RESCAN_MS + PROBE_MS + LEARN_MS + siteNetworks.length * SCAN_EVERY_MS + SCAN_MS);

  test('only the learned summary switched on, every site network swept', async ({ page }) => {
    const { request } = page;
    expect(siteRoute).not.toBe('');
    expect(gateway).not.toBe('');
    expect(core).not.toBe('');
    expect(siteNetworks.length).toBeGreaterThan(1);

    // Nothing inside the site is reachable to a sweep yet, so nothing of it
    // may be known: otherwise the steps below would prove nothing.
    const before = await discoveredIPs(request);
    expect(
      siteNetworks.flatMap((cidr) => siteDevices.get(cidr) ?? []).filter((ip) => before.has(ip)),
      'site devices discovered before the summary was switched on',
    ).toEqual([]);
    expect(
      await pending(siteNetworks, request, (s) => s === undefined),
      'site networks listed at start',
    ).toEqual([]);

    const saved = await request.post('/api/v1/device-credentials', {
      headers: { 'X-CSRF-Token': await csrfToken(request) },
      data: { name: 'discovery-first-run', community },
    });
    expect(saved.status(), 'POST /api/v1/device-credentials').toBe(200);
    const savedAt = Date.now();

    await test.step('the host route to the site is learned, switched off', async () => {
      await expect
        .poll(
          async () => {
            const route = (await subnets(request)).find((s) => s.cidr === siteRoute);
            return route === undefined
              ? 'absent'
              : { learned: route.learned, enabled: route.enabled };
          },
          { message: `${siteRoute} as a learned target`, timeout: RESCAN_MS, intervals: [2_000] },
        )
        .toEqual({ learned: true, enabled: false });
    });
    const routeAfter = Date.now() - savedAt;

    const summaryName = (await subnets(request)).find((s) => s.cidr === siteRoute)?.name ?? '';
    const switched = await request.put('/api/v1/security/devices/subnets', {
      headers: { 'X-CSRF-Token': await csrfToken(request) },
      data: { cidr: siteRoute, name: summaryName, enabled: true },
    });
    expect(switched.status(), `PUT /api/v1/security/devices/subnets ${siteRoute}`).toBe(200);
    const enabledAt = Date.now();
    const scan = scanner(request);

    await test.step('probing the summary finds the core switch inside it', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            return (await discoveredIPs(request)).has(core);
          },
          { message: `${core} discovered`, timeout: PROBE_MS, intervals: [2_000] },
        )
        .toBe(true);
    });
    const coreAfter = Date.now() - enabledAt;

    await test.step('the site networks are learned from it, switched off', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            return pending(siteNetworks, request, (s) => s?.learned === true && !s.enabled);
          },
          {
            message: 'site networks not listed as learned and switched off',
            timeout: LEARN_MS,
            intervals: [2_000],
          },
        )
        .toEqual([]);
    });

    await test.step('sweeping the summary finds devices in every site network', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            const found = await discoveredIPs(request);
            return siteNetworks.filter(
              (cidr) => !(siteDevices.get(cidr) ?? []).some((ip) => found.has(ip)),
            );
          },
          {
            message: 'site networks with no device discovered',
            timeout: siteNetworks.length * SCAN_EVERY_MS + SCAN_MS,
            intervals: [2_000],
          },
        )
        .toEqual([]);
    });

    const listed = await subnets(request);
    expect(
      listed.filter((s) => s.enabled).map((s) => s.cidr),
      'target networks switched on',
    ).toEqual([siteRoute]);

    test.info().annotations.push({
      type: 'learned summary sweep',
      description: `${siteRoute} learned ${routeAfter} ms after the credential was saved; ${core} found ${coreAfter} ms after it was switched on; devices in all ${siteNetworks.length} site networks ${Date.now() - enabledAt} ms after it was switched on`,
    });
  });
});
