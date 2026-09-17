import type { Meta, StoryObj } from '@storybook/react-vite';
import type { TopologyLink, TopologyNode } from '../types/topology';
import { TopologyGraph } from './TopologyGraph';

/**
 * The map is where the a11y gate earns its keep: every node is a control, so
 * each one needs a name, a focus ring and a target big enough to hit. These
 * stories put a real-shaped fixture in front of axe.
 */
const meta = {
  title: 'Archetypes/TopologyGraph',
  component: TopologyGraph,
  parameters: { layout: 'padded' },
} satisfies Meta<typeof TopologyGraph>;

export default meta;

type Story = StoryObj<typeof meta>;

function node(id: string, label: string, deviceType: string): TopologyNode {
  return {
    id,
    clientId: 'default',
    identityHash: id,
    displayName: label,
    deviceType,
    chassisId: '',
    sysName: label,
    primaryMac: '',
    primaryIp: '',
    firstSeen: '',
    lastSeen: '2026-09-17T06:00:00Z',
    metadata: {},
  };
}

function link(id: string, source: string, target: string, linkType: string): TopologyLink {
  return {
    id,
    sourceNodeId: source,
    targetNodeId: target,
    sourceInterface: '',
    targetInterface: '',
    linkType,
    status: 'up',
    speedMbps: 1000,
    utilizationPct: 0,
    firstSeen: '',
    lastSeen: '2026-09-17T06:00:00Z',
    evidence: {},
  };
}

const nodes: TopologyNode[] = [
  node('edge-01', 'edge-01', 'router'),
  node('fw-01', 'fw-01', 'firewall'),
  node('wlc-01', 'wlc-01', 'wireless-controller'),
  node('acc-01', 'acc-01', 'switch'),
  node('acc-02', 'acc-02', 'switch'),
  node('ap-01', 'ap-01', 'access-point'),
  node('ap-02', 'ap-02', 'access-point'),
  node('srv-01', 'srv-01', 'server'),
  node('ws-114', 'ws-114', 'unknown'),
];

const links: TopologyLink[] = [
  link('l1', 'edge-01', 'fw-01', 'lldp'),
  link('l2', 'fw-01', 'acc-01', 'lldp'),
  link('l3', 'acc-01', 'acc-02', 'cdp'),
  link('l4', 'acc-01', 'wlc-01', 'lldp'),
  link('l5', 'acc-02', 'ap-01', 'lldp'),
  link('l6', 'acc-02', 'ap-02', 'lldp'),
  link('l7', 'acc-01', 'srv-01', 'fdb'),
  link('l8', 'acc-02', 'ws-114', 'fdb'),
];

export const Default: Story = {
  args: { nodes, links, selectedId: '', onSelect: () => {} },
};

export const Selected: Story = {
  args: { nodes, links, selectedId: 'acc-01', onSelect: () => {} },
};

export const Empty: Story = {
  args: { nodes: [], links: [], selectedId: '', onSelect: () => {} },
};
