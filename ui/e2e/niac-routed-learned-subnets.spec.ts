import { type APIRequestContext, expect, test } from '@playwright/test';

/**
 * Site networks behind the edge router are learned, not typed (seed#2695).
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
 *   SEED_E2E_TYPED_NETWORK   the one site network entered by hand
 *   SEED_E2E_CORE            the agent in it whose address table names the rest
 *
 * The edge router names the site only as its summary route, and a learned
 * summary is not something the sweep can cover (it caps a target at 254
 * hosts), so the operator enters one site network. Every other one has to be
 * learned from the SNMP tables of a device inside it: the host has no route
 * narrower than the summary, so no site network can come from anywhere else.
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

const enabled = process.env.SEED_E2E_NIAC_ROUTED === '1';
const community = process.env.SEED_E2E_SNMP_COMMUNITY ?? '';
const gateway = process.env.SEED_E2E_GATEWAY ?? '';
const siteRoute = process.env.SEED_E2E_SITE_ROUTE ?? '';
const typedNetwork = process.env.SEED_E2E_TYPED_NETWORK ?? '';
const core = process.env.SEED_E2E_CORE ?? '';
const siteNetworks = (process.env.SEED_E2E_SITE_NETWORKS ?? '').split(',').filter(Boolean);
const learnedNetworks = siteNetworks.filter((cidr) => cidr !== typedNetwork);
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
  test.setTimeout(RESCAN_MS + SCAN_MS + LEARN_MS + SCAN_MS + 60_000);

  test('one site network typed in, the rest learned from SNMP and swept', async ({ page }) => {
    const { request } = page;
    expect(siteRoute).not.toBe('');
    expect(gateway).not.toBe('');
    expect(core).not.toBe('');
    expect(learnedNetworks.length).toBeGreaterThan(0);
    expect(learnedNetworks.length).toBe(siteNetworks.length - 1);

    // Nothing inside the site is reachable to a sweep yet, so nothing of it
    // may be known: otherwise the steps below would prove nothing.
    const before = await discoveredIPs(request);
    expect(
      siteNetworks.flatMap((cidr) => siteDevices.get(cidr) ?? []).filter((ip) => before.has(ip)),
      'site devices discovered before any site network was entered',
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

    const typed = await request.post('/api/v1/security/devices/subnets', {
      headers: { 'X-CSRF-Token': await csrfToken(request) },
      data: { cidr: typedNetwork, name: 'site management', enabled: true },
    });
    expect(typed.status(), `POST /api/v1/security/devices/subnets ${typedNetwork}`).toBe(200);
    const typedAt = Date.now();
    const scan = scanner(request);

    await test.step('scanning the typed network finds the device inside it', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            return (await discoveredIPs(request)).has(core);
          },
          { message: `${core} discovered`, timeout: SCAN_MS, intervals: [2_000] },
        )
        .toBe(true);
    });

    await test.step('every other site network is learned from that device, switched off', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            return pending(learnedNetworks, request, (s) => s?.learned === true && !s.enabled);
          },
          {
            message: 'site networks not listed as learned and switched off',
            timeout: LEARN_MS,
            intervals: [2_000],
          },
        )
        .toEqual([]);
    });
    const learnedAfter = Date.now() - typedAt;

    const listed = await subnets(request);
    const nameOf = (cidr: string): string => listed.find((s) => s.cidr === cidr)?.name ?? '';
    // The name records which table named the network, and the host's own
    // table has nothing narrower than the summary.
    expect(
      learnedNetworks.filter(
        (cidr) => !/\((interface addresses|routing table)\)$/.test(nameOf(cidr)),
      ),
      `learned site networks not attributed to an SNMP table: ${learnedNetworks.map((c) => `${c} "${nameOf(c)}"`).join('; ')}`,
    ).toEqual([]);
    expect(
      listed.find((s) => s.cidr === typedNetwork)?.learned,
      `${typedNetwork} stays the operator's own entry`,
    ).toBe(false);

    const token = await csrfToken(request);
    for (const cidr of learnedNetworks) {
      const response = await request.put('/api/v1/security/devices/subnets', {
        headers: { 'X-CSRF-Token': token },
        data: { cidr, name: nameOf(cidr), enabled: true },
      });
      expect(response.status(), `PUT /api/v1/security/devices/subnets ${cidr}`).toBe(200);
    }
    const enabledAt = Date.now();

    await test.step('switching them on finds devices in every one', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            const found = await discoveredIPs(request);
            return learnedNetworks.filter(
              (cidr) => !(siteDevices.get(cidr) ?? []).some((ip) => found.has(ip)),
            );
          },
          {
            message: 'learned site networks with no device discovered',
            timeout: SCAN_MS,
            intervals: [2_000],
          },
        )
        .toEqual([]);
    });

    test.info().annotations.push({
      type: 'learned target networks',
      description: `${siteRoute} learned ${routeAfter} ms after the credential was saved; ${learnedNetworks.length} site networks learned ${learnedAfter} ms after ${typedNetwork} was entered; their devices ${Date.now() - enabledAt} ms after they were switched on: ${learnedNetworks.map((c) => `${c} "${nameOf(c)}"`).join('; ')}`,
    });
  });
});
