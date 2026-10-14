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
 * rescan from the save. A target is polled as soon as it is registered, and
 * the topology reconcilers turn a poll into a node on their own 30 s tick.
 *
 * The last test adds one more target by hand, the way an operator would: v2c,
 * the saved community picked from the vault, left disabled so the scheduler
 * never polls it. "Test target" then has to produce its first successful
 * poll on its own (seed#2523).
 *
 * A node is built from the target's polled sysinfo and carries no address
 * until a later ARP pass backfills one, so it is matched on sysName: the name
 * promotion gave the target.
 */
const RESCAN_MS = 60_000;
const PROMOTION_MS = 3 * RESCAN_MS;
const POLL_MS = 30_000;
const RECONCILE_MS = 2 * 30_000;

const enabled = process.env.SEED_E2E_NIAC_LINK === '1';
const link = process.env.SEED_E2E_LINK ?? '';
const agentIPs = (process.env.SEED_E2E_SNMP_IPS ?? '').split(',').filter(Boolean);
const community = process.env.SEED_E2E_SNMP_COMMUNITY ?? '';

test.skip(
  !enabled || agentIPs.length === 0,
  'needs scripts/e2e-niac-link.sh with a scenario that runs SNMP agents',
);

interface PollingTarget {
  name: string;
  ipAddress: string;
  enabled: boolean;
  credentialsId: string;
  id: string;
  lastStatus: string;
  lastPolledAt?: string;
  lastSuccessAt?: string;
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

async function topologySysNames(request: APIRequestContext): Promise<Set<string>> {
  const response = await request.get('/api/v1/topology/nodes');
  if (!response.ok()) {
    return new Set();
  }
  const { nodes } = (await response.json()) as { nodes?: { sysName?: string }[] };
  return new Set((nodes ?? []).map((node) => node.sysName ?? ''));
}

test.describe('promotion over a NIAC link', () => {
  test.setTimeout(PROMOTION_MS + POLL_MS + RECONCILE_MS + 30_000);

  test('the active interface is listed where the UI selects it', async ({ page }) => {
    const response = await page.request.get('/api/v1/interfaces');
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
        {
          message: 'targets with no successful poll recorded',
          timeout: POLL_MS,
          intervals: [1_000],
        },
      )
      .toEqual([]);

    const sysNames = (await pollingTargets(request))
      .filter((t) => agentIPs.includes(t.ipAddress))
      .map((t) => t.name);
    expect(sysNames.filter(Boolean), 'promoted targets named by sysName').toHaveLength(
      agentIPs.length,
    );
    await expect
      .poll(
        async () => {
          const nodes = await topologySysNames(request);
          return sysNames.filter((name) => !nodes.has(name));
        },
        { message: 'agents missing from the topology', timeout: RECONCILE_MS, intervals: [1_000] },
      )
      .toEqual([]);

    test.info().annotations.push({
      type: 'promotion',
      description: `${agentIPs.length} agents promoted ${promotedAfter} ms after the credential was saved, polled and in the topology ${Date.now() - savedAt} ms after`,
    });
  });

  test('a hand-added v2c target polls the agent on Test target', async ({ page }) => {
    const name = 'hand-added-v2c';
    await page.goto('/polling-targets');
    await page.getByTestId('target-add').click();
    const dialog = page.getByRole('dialog', { name: 'Add polling target' });
    await dialog.getByLabel('Name', { exact: true }).fill(name);
    await dialog.getByLabel('IP address', { exact: true }).fill(agentIPs[0] ?? '');
    await dialog.getByLabel('SNMP version', { exact: true }).selectOption('v2c');
    await dialog
      .getByLabel('Credential', { exact: true })
      .selectOption({ label: 'discovery-first-run' });
    await dialog.getByRole('button', { name: 'Add target' }).click();
    await expect(dialog).toBeHidden();

    const added = (await pollingTargets(page.request)).find((t) => t.name === name);
    expect(added, 'the hand-added target').toBeDefined();
    expect(added?.enabled, 'a new target saves disabled').toBe(false);
    expect(added?.lastPolledAt, 'nothing polled the disabled target').toBeUndefined();

    await page.getByTestId(`target-row-${added?.id}`).click();
    await page.getByTestId('target-test').click();

    await expect(page.getByTestId('target-last-status')).toContainText('ok', { timeout: 25_000 });
    await expect(page.getByTestId('target-last-success')).not.toContainText('Never');
    const polled = (await pollingTargets(page.request)).find((t) => t.name === name);
    expect(polled?.lastStatus).toBe('ok');
    expect(polled?.lastSuccessAt, 'lastSuccessAt after a successful test poll').toBeTruthy();
  });
});
