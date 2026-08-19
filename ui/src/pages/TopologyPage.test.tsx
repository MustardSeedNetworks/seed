/**
 * TopologyPage had no coverage at all — no unit test, no E2E spec — which is
 * why these come before the archetype refactor rather than after it. The page
 * is List + detail in shape despite its name, so the rule under test is the
 * same one PollingTargetsPage locks: anything the poller has not measured must
 * read as unmeasured, never as healthy.
 *
 * The interface status pair is where that rule actually bites. RFC 2233 defines
 * ifOperStatus 1..7; only 1 is up and 2 is down. A `testing`, `dormant` or
 * `unknown` interface is NOT down and NOT up, and rendering it green would be a
 * confident lie about a port nobody has confirmed.
 */
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  TopologyInterface,
  TopologyLink,
  TopologyNode,
  TopologyNodeDetailResponse,
} from '../types/topology';

function node(over: Partial<TopologyNode>): TopologyNode {
  return {
    id: 'n1',
    clientId: 'c',
    identityHash: 'h',
    displayName: '',
    deviceType: '',
    chassisId: '',
    sysName: '',
    primaryMac: '',
    primaryIp: '',
    firstSeen: '',
    lastSeen: '',
    metadata: {},
    ...over,
  };
}

function iface(over: Partial<TopologyInterface>): TopologyInterface {
  return {
    id: 1,
    nodeId: 'core',
    ifIndex: 1,
    ifName: 'Gi0/1',
    ifDescr: '',
    ifAlias: '',
    ifType: 6,
    ifAdminStatus: 1,
    ifOperStatus: 1,
    ifPhysAddr: '',
    speedBps: 0,
    lastSeen: '',
    ...over,
  };
}

function link(over: Partial<TopologyLink>): TopologyLink {
  return {
    id: 'l1',
    sourceNodeId: 'core',
    targetNodeId: 'edge',
    sourceInterface: 'Gi0/1',
    targetInterface: 'Gi0/2',
    linkType: 'lldp',
    status: '',
    speedMbps: 0,
    utilizationPct: 0,
    firstSeen: '',
    lastSeen: '',
    evidence: {},
    ...over,
  };
}

const nodes: TopologyNode[] = [
  node({ id: 'core', displayName: 'core-01', deviceType: 'cisco', primaryIp: '10.44.10.2' }),
  // No displayName and no deviceType: the row must fall back to sysName and
  // the badge to 'n/a' rather than rendering an empty chip.
  node({ id: 'bare', sysName: 'sw-bare' }),
];

const detail: TopologyNodeDetailResponse = {
  node: node({
    id: 'core',
    displayName: 'core-01',
    deviceType: 'cisco',
    sysName: 'core-01.msn.lab',
    primaryMac: '00:11:22:33:44:55',
    primaryIp: '10.44.10.2',
    firstSeen: '',
    lastSeen: '',
  }),
  interfaces: [
    iface({ id: 1, ifIndex: 1, ifName: 'Gi0/1', ifOperStatus: 1, speedBps: 1_000_000_000 }),
    iface({ id: 2, ifIndex: 2, ifName: 'Gi0/2', ifOperStatus: 2, speedBps: 100_000_000 }),
    // ifOperStatus 4 = unknown. Neither up nor down.
    iface({ id: 3, ifIndex: 3, ifName: 'Gi0/3', ifOperStatus: 4, speedBps: 0 }),
  ],
  links: [
    link({ id: 'l1', sourceNodeId: 'core', targetNodeId: 'edge-77' }),
    // Reversed direction — the far end is the SOURCE here.
    link({ id: 'l2', sourceNodeId: 'spine-9', targetNodeId: 'core' }),
  ],
};

const state = {
  nodes: nodes as TopologyNode[],
  listError: null as string | null,
  listLoading: false,
  detail: detail as TopologyNodeDetailResponse | null,
  detailError: null as string | null,
  detailLoading: false,
};

vi.mock('../hooks/useTopology', () => ({
  useTopologyNodes: () => ({
    nodes: state.nodes,
    loading: state.listLoading,
    error: state.listError,
    refresh: vi.fn(),
  }),
  useTopologyNode: (id: string) => ({
    detail: id ? state.detail : null,
    loading: id ? state.detailLoading : false,
    error: id ? state.detailError : null,
    refresh: vi.fn(),
  }),
}));

const { TopologyPage } = await import('./TopologyPage');

function reset(): void {
  state.nodes = nodes;
  state.listError = null;
  state.listLoading = false;
  state.detail = detail;
  state.detailError = null;
  state.detailLoading = false;
}

describe('TopologyPage — honest state', () => {
  beforeEach(reset);

  it('does not colour an interface green unless it is genuinely up', async () => {
    render(<TopologyPage />);
    await userEvent.click(screen.getByTestId('node-row-core'));

    const rows = screen.getAllByRole('row').filter((r) => within(r).queryByText(/^Gi0\//));
    const operOf = (name: string): Element => {
      const row = rows.find((r) => within(r).queryByText(name));
      if (!row) throw new Error(`no row for ${name}`);
      const oper = within(row).getByText('oper');
      return oper;
    };

    expect(operOf('Gi0/1').className).toContain('text-status-success');
    expect(operOf('Gi0/2').className).toContain('text-status-error');
    // The whole point: 'unknown' is muted, not green and not red.
    const unknown = operOf('Gi0/3').className;
    expect(unknown).toContain('text-text-muted');
    expect(unknown).not.toContain('text-status-success');
    expect(unknown).not.toContain('text-status-error');
  });

  it('prints an em dash for a speed it does not know, not a zero', async () => {
    render(<TopologyPage />);
    await userEvent.click(screen.getByTestId('node-row-core'));

    expect(screen.getByText('1.0 Gbps')).toBeTruthy();
    expect(screen.getByText('100 Mbps')).toBeTruthy();
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    expect(screen.queryByText('0 bps')).toBeNull();
  });

  it("says 'never' for an absent timestamp rather than inventing an epoch date", async () => {
    render(<TopologyPage />);
    await userEvent.click(screen.getByTestId('node-row-core'));

    // firstSeen and lastSeen are both empty on this fixture.
    expect(screen.getAllByText('never')).toHaveLength(2);
    expect(screen.queryByText(/1970/)).toBeNull();
  });
});

describe('TopologyPage — list and selection', () => {
  beforeEach(reset);

  it('shows nothing selected until a node is clicked', () => {
    render(<TopologyPage />);

    expect(screen.getByText('Select a node to see interfaces and links.')).toBeTruthy();
  });

  it('falls back to sysName and an n/a badge when a node has no display name', () => {
    render(<TopologyPage />);
    const row = within(screen.getByTestId('node-row-bare'));

    expect(row.getByText('sw-bare')).toBeTruthy();
    expect(row.getByText('n/a')).toBeTruthy();
  });

  it('clears the selection back to the empty state', async () => {
    render(<TopologyPage />);
    await userEvent.click(screen.getByTestId('node-row-core'));
    expect(screen.getByRole('heading', { name: 'core-01' })).toBeTruthy();

    await userEvent.click(screen.getByRole('button', { name: 'Clear' }));

    expect(screen.getByText('Select a node to see interfaces and links.')).toBeTruthy();
  });

  it('names the far end of a link regardless of which way it was recorded', async () => {
    render(<TopologyPage />);
    await userEvent.click(screen.getByTestId('node-row-core'));

    expect(screen.getByText('↔ edge-77')).toBeTruthy();
    expect(screen.getByText('↔ spine-9')).toBeTruthy();
    expect(screen.queryByText('↔ core')).toBeNull();
  });

  it('counts nodes with correct pluralisation', () => {
    state.nodes = [nodes[0]];
    render(<TopologyPage />);

    expect(screen.getByText('1 node')).toBeTruthy();
  });
});

describe('TopologyPage — archetype rules', () => {
  beforeEach(reset);

  // The shared parts enforce that a record's colour comes from its state and
  // never its category. The old page coloured a vendor badge from a cat-* ramp,
  // which read as status to anyone scanning the column.
  it('does not colour a row by vendor', () => {
    render(<TopologyPage />);

    const row = screen.getByTestId('node-row-core');
    expect(row.className).not.toMatch(/cat-/);
    expect(row.innerHTML).not.toMatch(/cat-/);
  });

  // A node the reconcilers have never dated is unmeasured, not healthy. Green
  // on a node that was never seen is exactly the reassuring-when-it-knows-
  // nothing failure the shell exists to prevent.
  it('marks a never-seen node unknown rather than ok', () => {
    state.nodes = [
      node({ id: 'seen', displayName: 'seen-01', lastSeen: '2026-08-19T10:00:00Z' }),
      node({ id: 'unseen', displayName: 'unseen-01', lastSeen: '' }),
    ];
    render(<TopologyPage />);

    const bar = (id: string) =>
      screen.getByTestId(`node-row-${id}`).querySelector('span[aria-hidden="true"]');
    expect(bar('seen')?.className).toContain('bg-status-success');
    expect(bar('unseen')?.className).toContain('bg-text-disabled');
  });
});

describe('TopologyPage — degraded backends', () => {
  beforeEach(reset);

  it('surfaces a list error instead of an empty list that looks like no devices', () => {
    state.nodes = [];
    state.listError = 'topology store unavailable';
    render(<TopologyPage />);

    expect(screen.getByText('topology store unavailable')).toBeTruthy();
    expect(screen.queryByText(/No nodes yet/)).toBeNull();
  });

  it('distinguishes an empty topology from a broken one', () => {
    state.nodes = [];
    render(<TopologyPage />);

    expect(screen.getByText(/No nodes yet/)).toBeTruthy();
  });

  it('surfaces a detail error rather than rendering a blank panel', async () => {
    render(<TopologyPage />);
    state.detail = null;
    state.detailError = 'node fetch failed';
    await userEvent.click(screen.getByTestId('node-row-core'));

    expect(screen.getByText('node fetch failed')).toBeTruthy();
  });

  it('says a node was not found when the detail comes back empty', async () => {
    render(<TopologyPage />);
    state.detail = null;
    await userEvent.click(screen.getByTestId('node-row-core'));

    expect(screen.getByText('Node not found.')).toBeTruthy();
  });
});
