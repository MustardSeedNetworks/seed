import { type APIRequestContext, expect, test } from '@playwright/test';

/**
 * Discovered SNMP agents become polled targets and topology nodes with no
 * target typed in by hand (seed#2692).
 *
 * Runs only under scripts/e2e-niac-link.sh, after first-run-niac-link.spec.ts
 * and against the same daemon, which the script started with nothing but its
 * interface, port and paths configured. The script exports:
 *
 *   SEED_E2E_NIAC_LINK       set to 1 to enable this spec
 *   SEED_E2E_LINK            seed's end of the veth pair
 *   SEED_E2E_SNMP_IPS        comma-separated addresses of the scenario's agents
 *   SEED_E2E_SNMP_COMMUNITY  the community every one of them answers
 *
 * The credential is saved the way the first-run SNMP step saves it. The
 * sweep after that one reaches the agents with it, and promotion reads a sweep
 * a round behind discovery (enumerate.Service.notifySweep), so the targets land
 * within two rescan cycles of that sweep: three cycles of the compiled-in 60 s
 * rescan from the save. A target is polled as soon as it is registered, so
 * the topology gets no separate allowance beyond one poll.
 */
const RESCAN_MS = 60_000;
const PROMOTION_MS = 3 * RESCAN_MS;
const POLL_MS = 30_000;

const enabled = process.env.SEED_E2E_NIAC_LINK === '1';
const link = process.env.SEED_E2E_LINK ?? '';
const agentIPs = (process.env.SEED_E2E_SNMP_IPS ?? '').split(',').filter(Boolean);
const community = process.env.SEED_E2E_SNMP_COMMUNITY ?? '';

test.skip(
  !enabled || agentIPs.length === 0,
  'needs scripts/e2e-niac-link.sh with a scenario that runs SNMP agents',
);

interface PollingTarget {
  ipAddress: string;
  enabled: boolean;
  credentialsId: string;
  lastStatus: string;
  lastPolledAt?: string;
}

async function csrfToken(request: APIRequestContext): Promise<string> {
  const response = await request.get('/api/v1/auth/csrf');
  expect(response.status(), 'GET /api/v1/auth/csrf').toBe(200);
  const { token } = (await response.json()) as { token: string };
  return token;
}

async function pollingTargets(request: APIRequestContext): Promise<PollingTarget[]> {
  const response = await request.get('/api/v1/polling-targets');
  if (!response.ok()) {
    return [];
  }
  const { targets } = (await response.json()) as { targets?: PollingTarget[] };
  return targets ?? [];
}

async function topologyNodeIPs(request: APIRequestContext): Promise<Set<string>> {
  const response = await request.get('/api/v1/topology/nodes');
  if (!response.ok()) {
    return new Set();
  }
  const { nodes } = (await response.json()) as { nodes?: { primaryIp?: string }[] };
  return new Set((nodes ?? []).map((node) => node.primaryIp ?? ''));
}

test.describe('promotion over a NIAC link', () => {
  test.setTimeout(PROMOTION_MS + POLL_MS + 30_000);

  test('the active interface is listed where the UI selects it', async ({ page }) => {
    const response = await page.request.get('/api/v1/interfaces?categorized=true');
    expect(response.status()).toBe(200);
    expect(JSON.stringify(await response.json())).toContain(`"${link}"`);
  });

  test('SNMP agents become polled targets and topology nodes', async ({ page }) => {
    const { request } = page;
    const saved = await request.post('/api/v1/device-credentials', {
      headers: { 'X-CSRF-Token': await csrfToken(request) },
      data: { name: 'discovery-first-run', community },
    });
    expect(saved.status(), 'POST /api/v1/device-credentials').toBe(200);
    const { id: credentialID } = (await saved.json()) as { id: string };
    const savedAt = Date.now();

    await expect
      .poll(
        async () => {
          const targets = await pollingTargets(request);
          return agentIPs.filter(
            (ip) =>
              !targets.some(
                (t) => t.ipAddress === ip && t.enabled && t.credentialsId === credentialID,
              ),
          );
        },
        {
          message: 'agents with no enabled target on the saved credential',
          timeout: PROMOTION_MS,
          intervals: [2_000],
        },
      )
      .toEqual([]);
    const promotedAfter = Date.now() - savedAt;

    await expect
      .poll(
        async () => {
          const targets = await pollingTargets(request);
          return agentIPs.filter(
            (ip) =>
              !targets.some((t) => t.ipAddress === ip && t.lastStatus === 'ok' && t.lastPolledAt),
          );
        },
        { message: 'targets with no successful poll recorded', timeout: POLL_MS, intervals: [1_000] },
      )
      .toEqual([]);

    await expect
      .poll(
        async () => {
          const nodes = await topologyNodeIPs(request);
          return agentIPs.filter((ip) => !nodes.has(ip));
        },
        { message: 'agents missing from the topology', timeout: POLL_MS, intervals: [1_000] },
      )
      .toEqual([]);

    test.info().annotations.push({
      type: 'promotion',
      description: `${agentIPs.length} agents promoted ${promotedAfter} ms after the credential was saved, polled and in the topology ${Date.now() - savedAt} ms after`,
    });
  });
});
