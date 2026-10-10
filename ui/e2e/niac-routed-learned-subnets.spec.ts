import { type APIRequestContext, expect, test } from '@playwright/test';
import type { PathResponse } from '../src/types/generated/path-response';
import type { SubnetDecisionRequest } from '../src/types/generated/subnet-decision-request';
import type { SubnetResponse as Subnet } from '../src/types/generated/subnet-response';

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
 * Every learned network is offered for the operator's decision and nothing is
 * swept until the operator adds it (seed#3108). The spec adds only the summary.
 *
 * The edge router names the site only as its summary route, which is wider
 * than one sweep may probe. With only that summary added, the sweep
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
const PROBED_HOSTS = 4;
const PROBE_MS = 6 * SCAN_EVERY_MS + SCAN_MS;
// A sweep of the transit /24 and one summary block takes longer than one press
// period (about 28 s on 2026-10-10), so only every other press starts one.
const SWEEP_MS = 2 * SCAN_EVERY_MS;
// The path route gives a trace two minutes.
const TRACE_MS = 120_000;

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

/** An IPv4 network contains addr: the path step's check on a hop's route. */
function covers(network: string, prefix: number, addr: string): boolean {
  const toInt = (ip: string) => ip.split('.').reduce((n, octet) => n * 256 + Number(octet), 0);
  if (prefix < 0 || prefix > 32) {
    return false;
  }
  const size = 2 ** (32 - prefix);
  return Math.floor(toInt(network) / size) === Math.floor(toInt(addr) / size);
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

/** The learned networks offered for the operator's decision. */
async function offered(request: APIRequestContext): Promise<Set<string>> {
  const response = await request.get('/api/v1/security/devices/subnets/pending');
  expect(response.status(), 'GET /api/v1/security/devices/subnets/pending').toBe(200);
  return new Set(((await response.json()) as Subnet[]).map((s) => s.cidr));
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
 * waiting on a sweep's result also drives the sweeps it waits on. Every
 * rate-limited route shares one 5-per-minute bucket per client, and the
 * specs that ran before this one on the same daemon may have spent part of
 * it, so a 429 is a press that did not land: the next one comes a period
 * later, and the poll's own timeout bounds the wait.
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
    expect([200, 429], 'POST /api/v1/security/devices/scan').toContain(response.status());
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
  test.setTimeout(
    RESCAN_MS + PROBE_MS + LEARN_MS + siteNetworks.length * SWEEP_MS + SCAN_MS + TRACE_MS,
  );

  test('only the learned summary switched on, its site networks swept', async ({ page }) => {
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

    await test.step('the host route to the site is learned and offered, switched off', async () => {
      await expect
        .poll(
          async () => {
            const route = (await subnets(request)).find((s) => s.cidr === siteRoute);
            return route === undefined
              ? 'absent'
              : {
                  learned: route.learned,
                  enabled: route.enabled,
                  offered: (await offered(request)).has(siteRoute),
                };
          },
          { message: `${siteRoute} as a learned target`, timeout: RESCAN_MS, intervals: [2_000] },
        )
        .toEqual({ learned: true, enabled: false, offered: true });
    });
    const routeAfter = Date.now() - savedAt;

    const decision: SubnetDecisionRequest = { cidr: siteRoute, decision: 'added' };
    const added = await request.post('/api/v1/security/devices/subnets/pending', {
      headers: { 'X-CSRF-Token': await csrfToken(request) },
      data: decision,
    });
    expect(added.status(), `POST /api/v1/security/devices/subnets/pending ${siteRoute}`).toBe(200);
    expect((await offered(request)).has(siteRoute), `${siteRoute} still offered once added`).toBe(
      false,
    );
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

    await test.step('the site networks are learned from it and offered, switched off', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            const offers = await offered(request);
            return pending(
              siteNetworks,
              request,
              (s) => s?.learned === true && !s.enabled && offers.has(s.cidr),
            );
          },
          {
            message: 'site networks not listed as learned, offered and switched off',
            timeout: LEARN_MS,
            intervals: [2_000],
          },
        )
        .toEqual([]);
    });

    // Every site network that holds a device must show one. The probe alone
    // reaches .1 to .4, so only a device above them shows its /24 was swept
    // whole; some site networks hold nothing above .4, and the rest must each
    // show one of those.
    const populated = siteNetworks.filter((cidr) => (siteDevices.get(cidr) ?? []).length > 0);
    const sweptOnly = new Map(
      siteNetworks
        .map(
          (cidr) =>
            [
              cidr,
              (siteDevices.get(cidr) ?? []).filter((ip) => Number(ip.split('.')[3]) > PROBED_HOSTS),
            ] as const,
        )
        .filter(([, ips]) => ips.length > 0),
    );
    expect(sweptOnly.size, `site networks with a device above .${PROBED_HOSTS}`).toBeGreaterThan(1);

    await test.step('sweeping the summary finds devices in every site network', async () => {
      await expect
        .poll(
          async () => {
            await scan();
            const found = await discoveredIPs(request);
            const missing = (ips: readonly string[]) => !ips.some((ip) => found.has(ip));
            return {
              noDevice: populated.filter((cidr) => missing(siteDevices.get(cidr) ?? [])),
              notSweptWhole: [...sweptOnly].filter(([, ips]) => missing(ips)).map(([cidr]) => cidr),
            };
          },
          {
            message: `site networks with no device, or none above .${PROBED_HOSTS}, discovered`,
            timeout: siteNetworks.length * SWEEP_MS + SCAN_MS,
            intervals: [2_000],
          },
        )
        .toEqual({ noDevice: [], notSweptWhole: [] });
    });

    const listed = await subnets(request);
    expect(
      listed.filter((s) => s.enabled).map((s) => s.cidr),
      'target networks switched on',
    ).toEqual([siteRoute]);

    // The path view reads each router's own table (seed#2587): a hop that a
    // discovered router answers on carries the route it holds for the target.
    const target = [...sweptOnly.values()][0]?.[0] ?? '';
    const routed = await test.step("the traced path carries the routers' own routes", async () => {
      const response = await request.post('/api/v1/path/path', {
        headers: { 'X-CSRF-Token': await csrfToken(request) },
        data: { source: 'self', destination: target, method: 'l3', protocol: 'icmp' },
      });
      expect(response.status(), `POST /api/v1/path/path ${target}`).toBe(200);
      const { l3Path } = (await response.json()) as PathResponse;
      const hops = (l3Path?.hops ?? []).filter((hop) => hop.route !== undefined);
      expect(hops.length, `hops toward ${target} carrying a route`).toBeGreaterThan(0);
      for (const { ip, route } of hops) {
        expect(
          covers(route?.destination ?? '', route?.prefix ?? -1, target),
          `${ip} route covers ${target}`,
        ).toBe(true);
        if (route?.type === 'remote') {
          expect(route.nextHop ?? '', `${ip} remote route's next hop`).not.toMatch(
            /^(0\.0\.0\.0)?$/,
          );
        }
      }
      return hops.map(
        ({ ip, route }) =>
          `${ip} via ${route?.device}: ${route?.destination}/${route?.prefix} -> ${route?.nextHop}`,
      );
    });

    test.info().annotations.push({
      type: 'learned summary sweep',
      description: `${siteRoute} learned ${routeAfter} ms after the credential was saved; ${siteNetworks.length} site networks learned, devices found in ${populated.join(', ')}; ${core} found ${coreAfter} ms after it was switched on; devices above .${PROBED_HOSTS} in ${[...sweptOnly.keys()].join(', ')} ${Date.now() - enabledAt} ms after it was switched on; path to ${target}: ${routed.join('; ')}`,
    });
  });
});
