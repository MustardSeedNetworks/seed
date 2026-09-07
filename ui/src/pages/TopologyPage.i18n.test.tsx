/**
 * TopologyPage.i18n.test.tsx — the topology page renders real locale copy.
 *
 * #1942, S1-14. `TopologyPage.test.tsx` already asserts the page's honesty
 * rules (unknown is not green, an absent speed is an em dash), and every one
 * of those assertions matched English hardcoded in the component, which is
 * exactly why wrecking both locale trees left them green.
 *
 * The pattern is the accepted bar's: assert in both locales, and assert that
 * no English is left behind under `es`. Standard protocol terms are excluded
 * on purpose — `admin`/`oper` are the RFC 2233 ifAdminStatus/ifOperStatus
 * fields, and `MAC`, `Gbps`, `LLDP` are names, not prose. Translating those
 * would make the screen harder to match against a switch CLI, not easier.
 */
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../i18n';
import type { TopologyNodeDetailResponse } from '../types/topology';

const detail: TopologyNodeDetailResponse = {
  node: {
    id: 'core',
    clientId: 'c',
    identityHash: 'h',
    displayName: 'core-01',
    deviceType: 'cisco',
    chassisId: '',
    sysName: 'core-01.msn.lab',
    primaryMac: '00:11:22:33:44:55',
    primaryIp: '10.44.10.2',
    firstSeen: '2026-01-02T03:04:05Z',
    // Empty on purpose: the "no timestamp" word is copy too, and it is the
    // one an operator most needs to read as "we have never seen this".
    lastSeen: '',
    metadata: {},
  },
  interfaces: [
    {
      id: 1,
      nodeId: 'core',
      ifIndex: 1,
      ifName: 'Gi0/1',
      ifDescr: '',
      ifAlias: '',
      ifType: 6,
      ifAdminStatus: 1,
      ifOperStatus: 1,
      ifPhysAddr: '00:11:22:33:44:66',
      speedBps: 1_000_000_000,
      lastSeen: '',
    },
  ],
  links: [
    {
      id: 'l1',
      sourceNodeId: 'core',
      targetNodeId: 'edge-77',
      sourceInterface: 'Gi0/1',
      targetInterface: 'Gi0/2',
      linkType: 'lldp',
      status: '',
      speedMbps: 0,
      utilizationPct: 0,
      firstSeen: '',
      lastSeen: '',
      evidence: {},
    },
  ],
};

const state = {
  detail: detail as TopologyNodeDetailResponse | null,
  nodes: [detail.node],
};

vi.mock('../hooks/useTopology', () => ({
  useTopologyNodes: () => ({
    nodes: state.nodes,
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
  useTopologyNode: (id: string) => ({
    detail: id ? state.detail : null,
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
}));

const { TopologyPage } = await import('./TopologyPage');

async function renderIn(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  render(<TopologyPage />);
}

beforeEach(() => {
  state.detail = detail;
  state.nodes = [detail.node];
});

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('TopologyPage — real locale copy', () => {
  it('renders the English list and detail', async () => {
    await renderIn('en');

    // List pane: the count, the refresh control's accessible name, and the
    // empty state are all copy.
    expect(screen.getByText('1 node')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeVisible();
    expect(screen.getByText('Select a node to see interfaces and links.')).toBeVisible();

    await userEvent.click(screen.getByTestId('node-row-core'));

    expect(screen.getByText('Selected node')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Clear' })).toBeVisible();
    for (const label of [
      'Device type',
      'Sys name',
      'Primary MAC',
      'Primary IP',
      'First seen',
      'Last seen',
    ]) {
      expect(screen.getByText(label)).toBeVisible();
    }
    expect(screen.getByText('Never')).toBeVisible();
    expect(screen.getByText('Interfaces (1)')).toBeVisible();
    expect(screen.getByText('Neighbor link (1)')).toBeVisible();
    expect(screen.getByRole('columnheader', { name: 'Index' })).toBeVisible();
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeVisible();
    expect(screen.getByRole('columnheader', { name: 'Speed' })).toBeVisible();
  });

  it('says "0 nodes" and names the fix when nothing has been polled', async () => {
    state.nodes = [];
    state.detail = null;
    await renderIn('en');

    expect(screen.getByText('0 nodes')).toBeVisible();
    expect(
      screen.getByText('No nodes yet. Add a polling target and wait a poll cycle.'),
    ).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn('es');

    expect(screen.getByText('1 nodo')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Actualizar' })).toBeVisible();
    expect(
      screen.getByText('Seleccione un nodo para ver las interfaces y los enlaces.'),
    ).toBeVisible();

    await userEvent.click(screen.getByTestId('node-row-core'));

    // Every English string the page can render in this state. A hardcoded
    // label survives changeLanguage and shows up here.
    for (const english of [
      'Selected node',
      'Clear',
      'Device type',
      'Sys name',
      'Primary MAC',
      'Primary IP',
      'First seen',
      'Last seen',
      'Never',
      'Neighbor link (1)',
      'Index',
      'Name',
      'Speed',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }
    expect(screen.queryByRole('button', { name: 'Refresh' })).toBeNull();

    // …and the Spanish is actually there, so this cannot pass by rendering
    // an empty pane.
    expect(screen.getByText('Nodo seleccionado')).toBeVisible();
    expect(screen.getByText('Tipo de dispositivo')).toBeVisible();
    expect(screen.getByText('Interfaces (1)')).toBeVisible();
    expect(screen.getByText('Enlace vecino (1)')).toBeVisible();
    expect(screen.getByText('Nunca')).toBeVisible();
  });

  it('keeps standard protocol terms untranslated under es', async () => {
    await renderIn('es');
    await userEvent.click(screen.getByTestId('node-row-core'));

    const table = screen.getByRole('table');
    // ifAdminStatus / ifOperStatus, not prose.
    expect(within(table).getByText('admin')).toBeVisible();
    expect(within(table).getByText('oper')).toBeVisible();
    expect(within(table).getByText('1.0 Gbps')).toBeVisible();
    expect(screen.getByRole('columnheader', { name: 'MAC' })).toBeVisible();
  });

  it('says "0 nodos" and names the fix under es', async () => {
    state.nodes = [];
    state.detail = null;
    await renderIn('es');

    expect(screen.getByText('0 nodos')).toBeVisible();
    expect(
      screen.queryByText('No nodes yet. Add a polling target and wait a poll cycle.'),
    ).toBeNull();
  });
});
