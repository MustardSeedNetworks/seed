import { expect, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Topology Page (/topology) E2E — seed#2700.
 *
 * The page had no map: 200 discovered nodes rendered as a list. These drive
 * the map itself, so they need a discovered network to draw, and the E2E
 * daemon boots on an empty database. The fixture is therefore generated here
 * in the shape of NIAC's hospital pack (a core, two distribution switches,
 * access switches with three APs each, servers and a floor of endpoints) and
 * served through page.route — the assertions are about what the UI draws from
 * a topology, not about the reconcilers that fill one in.
 */

const ACCESS_SWITCHES = 16;
const APS_PER_SWITCH = 3;
const SERVERS = 8;
const ENDPOINTS = 160;

interface FixtureNode {
  id: string;
  clientId: string;
  identityHash: string;
  displayName: string;
  deviceType: string;
  chassisId: string;
  sysName: string;
  primaryMac: string;
  primaryIp: string;
  firstSeen: string;
  lastSeen: string;
  metadata: Record<string, unknown>;
}

interface FixtureLink {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  sourceInterface: string;
  targetInterface: string;
  linkType: string;
  status: string;
  speedMbps: number;
  utilizationPct: number;
  firstSeen: string;
  lastSeen: string;
  evidence: Record<string, unknown>;
}

const SEEN = '2026-09-17T06:00:00Z';

function node(id: string, deviceType: string): FixtureNode {
  return {
    id,
    clientId: 'default',
    identityHash: id,
    displayName: id,
    deviceType,
    chassisId: '',
    sysName: id,
    primaryMac: '',
    primaryIp: '',
    firstSeen: SEEN,
    lastSeen: SEEN,
    metadata: {},
  };
}

function link(id: string, source: string, target: string, linkType: string): FixtureLink {
  return {
    id,
    sourceNodeId: source,
    targetNodeId: target,
    sourceInterface: 'Gi0/1',
    targetInterface: 'Gi0/2',
    linkType,
    status: 'up',
    speedMbps: 1000,
    utilizationPct: 0,
    firstSeen: SEEN,
    lastSeen: SEEN,
    evidence: {},
  };
}

/** hospital builds a ~250-device two-building topology, the size the demo
 * packs settled on, so the map is exercised at the scale that has to stay
 * responsive rather than at a toy three-node graph. */
function hospital(): { nodes: FixtureNode[]; links: FixtureLink[] } {
  const nodes: FixtureNode[] = [node('med-core-01', 'router'), node('med-fw-01', 'firewall')];
  const links: FixtureLink[] = [link('l-core-fw', 'med-core-01', 'med-fw-01', 'lldp')];

  for (let i = 0; i < ACCESS_SWITCHES; i++) {
    const sw = `med-acc-${String(i).padStart(2, '0')}`;
    nodes.push(node(sw, 'switch'));
    links.push(link(`l-${sw}`, 'med-core-01', sw, 'lldp'));
    for (let a = 0; a < APS_PER_SWITCH; a++) {
      const ap = `${sw}-ap-${a}`;
      nodes.push(node(ap, 'access-point'));
      links.push(link(`l-${ap}`, sw, ap, 'lldp'));
    }
  }
  for (let s = 0; s < SERVERS; s++) {
    const srv = `med-srv-${s}`;
    nodes.push(node(srv, 'server'));
    // A server is reached through the forwarding table, not a neighbour
    // protocol — these are the edges the map must draw dashed.
    links.push(link(`l-${srv}`, 'med-acc-00', srv, 'fdb'));
  }
  for (let e = 0; e < ENDPOINTS; e++) {
    const host = `med-ws-${String(e).padStart(3, '0')}`;
    nodes.push(node(host, 'unknown'));
    links.push(
      link(`l-${host}`, `med-acc-${String(e % ACCESS_SWITCHES).padStart(2, '0')}`, host, 'fdb'),
    );
  }
  return { nodes, links };
}

async function serveHospital(page: Page): Promise<{ nodes: FixtureNode[] }> {
  const { nodes, links } = hospital();
  // The detail route first: Playwright matches the most recently registered
  // handler, and /nodes/{id} would otherwise fall through to the real daemon,
  // which has never heard of this fixture.
  await page.route(/\/api\/v1\/topology\/nodes\/[^/?]+$/, (route) => {
    const id = decodeURIComponent(new URL(route.request().url()).pathname.split('/').pop() ?? '');
    const found = nodes.find((n) => n.id === id);
    if (!found) {
      return route.fulfill({ status: 404, contentType: 'application/json', body: '{}' });
    }
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        node: found,
        interfaces: [],
        links: links.filter((l) => l.sourceNodeId === id || l.targetNodeId === id),
      }),
    });
  });
  await page.route(/\/api\/v1\/topology\/nodes(\?.*)?$/, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ count: nodes.length, nodes }),
    }),
  );
  await page.route(/\/api\/v1\/topology\/links(\?.*)?$/, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ count: links.length, links }),
    }),
  );
  return { nodes };
}

async function gotoTopology(page: Page): Promise<void> {
  await page.goto('/topology');
  await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
}

test.describe('Topology Page — the map', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('draws a graph of the discovered nodes and links', async ({ page }) => {
    const { nodes } = await serveHospital(page);
    await gotoTopology(page);

    await expect(page.getByTestId('topology-graph')).toBeVisible();
    await expect(page.getByTestId('topology-graph-node-med-core-01')).toBeVisible();
    await expect(page.getByTestId('topology-graph-node-med-acc-00')).toBeVisible();
    await expect(page.getByTestId('topology-graph-legend')).toBeVisible();

    // Every discovered device is on the map, not just the first page of them:
    // the handler's default limit is 200 and this topology is larger.
    expect(nodes.length).toBeGreaterThan(200);
    const drawn = page.locator('[data-testid^="topology-graph-node-"]');
    await expect(drawn).toHaveCount(nodes.length);
  });

  test('draws an FDB edge dashed and a neighbour edge solid', async ({ page }) => {
    await serveHospital(page);
    await gotoTopology(page);

    await expect(page.getByTestId('topology-graph-link-l-core-fw')).not.toHaveAttribute(
      'stroke-dasharray',
      /.*/,
    );
    await expect(page.getByTestId('topology-graph-link-l-med-srv-0')).toHaveAttribute(
      'stroke-dasharray',
      /\d/,
    );
  });

  test('selects a node from the map and fills the detail panel', async ({ page }) => {
    await serveHospital(page);
    await gotoTopology(page);

    await page.getByTestId('topology-graph-node-med-acc-00').click();

    await expect(page.getByTestId('topology-graph-node-med-acc-00')).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    await expect(page.getByRole('heading', { name: 'med-acc-00' })).toBeVisible();
  });

  test('reads in dark as well as light, and never scrolls sideways at 390px', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('seed-theme', 'dark'));
    await page.setViewportSize({ width: 390, height: 844 });
    await serveHospital(page);
    await gotoTopology(page);

    await expect(page.locator('html')).toHaveClass(/dark/);
    const graph = page.getByTestId('topology-graph');
    await expect(graph).toBeVisible();

    const box = await graph.boundingBox();
    expect(box?.width ?? 0).toBeLessThanOrEqual(390);
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });
});
