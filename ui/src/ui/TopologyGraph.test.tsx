/**
 * The map is the page's answer to "what does this network look like", so the
 * rules under test are the ones an operator reads off it without being told:
 * an FDB edge is a guess and must not look like an LLDP neighbour, a node is
 * a control and must work from the keyboard, and a graph with nothing in it
 * must say so rather than showing an empty box.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { TopologyLink, TopologyNode } from '../types/topology';
import { TopologyGraph } from './TopologyGraph';

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

function link(over: Partial<TopologyLink>): TopologyLink {
  return {
    id: 'l1',
    sourceNodeId: 'core',
    targetNodeId: 'acc',
    sourceInterface: '',
    targetInterface: '',
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
  node({ id: 'core', displayName: 'core-01', deviceType: 'router' }),
  node({ id: 'acc', displayName: 'acc-01', deviceType: 'switch' }),
  node({ id: 'host', sysName: 'ws-7', deviceType: 'unknown' }),
];

describe('TopologyGraph', () => {
  it('draws one element per node and per drawable link', () => {
    render(
      <TopologyGraph
        nodes={nodes}
        links={[link({ id: 'l1' }), link({ id: 'l2', sourceNodeId: 'acc', targetNodeId: 'host' })]}
        linksError={null}
        selectedId=""
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByTestId('topology-graph')).toBeTruthy();
    for (const id of ['core', 'acc', 'host']) {
      expect(screen.getByTestId(`topology-graph-node-${id}`)).toBeTruthy();
    }
    expect(screen.getByTestId('topology-graph-link-l1')).toBeTruthy();
    expect(screen.getByTestId('topology-graph-link-l2')).toBeTruthy();
  });

  it('dashes an FDB edge and leaves a neighbour edge solid — a guess must not look like a fact', () => {
    render(
      <TopologyGraph
        nodes={nodes}
        links={[
          link({ id: 'neighbour', linkType: 'lldp' }),
          link({ id: 'learned', linkType: 'fdb', sourceNodeId: 'acc', targetNodeId: 'host' }),
        ]}
        linksError={null}
        selectedId=""
        onSelect={vi.fn()}
      />,
    );

    expect(
      screen.getByTestId('topology-graph-link-learned').getAttribute('stroke-dasharray'),
    ).toBeTruthy();
    expect(
      screen.getByTestId('topology-graph-link-neighbour').getAttribute('stroke-dasharray'),
    ).toBeNull();
  });

  it('selects a node on click and on Enter, because the map is not mouse-only', async () => {
    const onSelect = vi.fn();
    render(
      <TopologyGraph
        nodes={nodes}
        links={[]}
        linksError={null}
        selectedId=""
        onSelect={onSelect}
      />,
    );

    await userEvent.click(screen.getByTestId('topology-graph-node-acc'));
    expect(onSelect).toHaveBeenCalledWith('acc');

    screen.getByTestId('topology-graph-node-core').focus();
    await userEvent.keyboard('{Enter}');
    expect(onSelect).toHaveBeenLastCalledWith('core');
  });

  it('marks the selected node pressed so it is not colour alone that says which one it is', () => {
    render(
      <TopologyGraph
        nodes={nodes}
        links={[]}
        linksError={null}
        selectedId="acc"
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByTestId('topology-graph-node-acc').getAttribute('aria-pressed')).toBe('true');
    expect(screen.getByTestId('topology-graph-node-core').getAttribute('aria-pressed')).toBe(
      'false',
    );
  });

  it('names each node for a screen reader with the label the list shows', () => {
    render(
      <TopologyGraph nodes={nodes} links={[]} linksError={null} selectedId="" onSelect={vi.fn()} />,
    );

    expect(screen.getByRole('button', { name: /core-01/ })).toBeTruthy();
    // No displayName: the label falls back to sysName, as the list row does.
    expect(screen.getByRole('button', { name: /ws-7/ })).toBeTruthy();
  });

  it('says the map is empty rather than drawing a blank frame', () => {
    render(
      <TopologyGraph nodes={[]} links={[]} linksError={null} selectedId="" onSelect={vi.fn()} />,
    );

    expect(screen.queryByTestId('topology-graph')).toBeNull();
    expect(screen.getByTestId('topology-graph-empty')).toBeTruthy();
  });

  it('renders a legend, because a dashed line means nothing unexplained', () => {
    render(
      <TopologyGraph nodes={nodes} links={[]} linksError={null} selectedId="" onSelect={vi.fn()} />,
    );

    expect(screen.getByTestId('topology-graph-legend')).toBeTruthy();
  });
});
