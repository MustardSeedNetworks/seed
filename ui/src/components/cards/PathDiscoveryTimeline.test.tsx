/**
 * PathDiscoveryTimeline tests — verify the unified L2+L3 timeline renders the
 * segments in path order (source -> L2 switches -> L3 routers -> destination),
 * surfaces an explicit empty state when there are no L2 hops, and toggles L2
 * port detail.
 */

import { fireEvent, render, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { PathResponse } from '../../types';
import { PATH_TIMELINE } from './PathDiscoveryTimeline';

const t = (_key: string, fallback: string): string => fallback;

function makeResult(overrides: Partial<PathResponse> = {}): PathResponse {
  return {
    l2Path: {
      hops: [
        {
          device: 'access-sw1',
          deviceIp: '10.0.0.2',
          ingressPort: {
            name: 'Gi0/1',
            index: 1,
            speed: '1Gbps',
            duplex: 'full',
            vlans: [10, 20],
            isTrunk: false,
            connectedTo: 'host',
          },
          egressPort: {
            name: 'Gi0/2',
            index: 2,
            speed: '1Gbps',
            duplex: 'full',
            vlans: [10],
            isTrunk: true,
            connectedTo: 'dist-sw1',
          },
          source: 'lldp',
        },
        {
          device: 'dist-sw1',
          deviceIp: '10.0.0.3',
          ingressPort: null,
          egressPort: null,
          source: 'cdp',
        },
      ],
    },
    l3Path: {
      target: '8.8.8.8',
      targetIp: '8.8.8.8',
      protocol: 'icmp',
      completed: true,
      hops: [
        { ttl: 1, ip: '192.168.1.1', rtt: 1_000_000, state: 'reply' },
        { ttl: 2, ip: '8.8.8.8', hostname: 'dns.google', rtt: 20_000_000, state: 'reply' },
      ],
    },
    ...overrides,
  };
}

describe('<PATH_TIMELINE>', () => {
  it('renders L2 switch hops before L3 router hops, ending at the destination', () => {
    const { getByTestId } = render(
      <PATH_TIMELINE
        result={makeResult()}
        maxRtt={20_000_000}
        expandedL2Hop={null}
        onToggleL2Hop={vi.fn()}
        t={t}
      />,
    );

    const l2First = getByTestId('l2-hop-0');
    const l2Second = getByTestId('l2-hop-1');
    const l3First = getByTestId('l3-hop-1');
    const dest = getByTestId('timeline-dest');

    // L2 segment precedes the L3 segment in document order.
    expect(
      l2First.compareDocumentPosition(l3First) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      l2First.compareDocumentPosition(l2Second) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    // Destination is last.
    expect(l3First.compareDocumentPosition(dest) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(dest.textContent).toContain('8.8.8.8');
  });

  it('shows an explicit empty state when L2 was requested but found no hops', () => {
    const { getByTestId, queryByTestId } = render(
      <PATH_TIMELINE
        result={makeResult({ l2Path: { hops: [] } })}
        maxRtt={20_000_000}
        expandedL2Hop={null}
        onToggleL2Hop={vi.fn()}
        t={t}
      />,
    );

    expect(getByTestId('l2-empty')).toBeTruthy();
    expect(queryByTestId('l2-hop-0')).toBeNull();
    // L3 segment still renders.
    expect(getByTestId('l3-hop-1')).toBeTruthy();
  });

  it('toggles L2 port detail via the hop header', () => {
    const onToggle = vi.fn();
    const { getByTestId } = render(
      <PATH_TIMELINE
        result={makeResult()}
        maxRtt={20_000_000}
        expandedL2Hop={null}
        onToggleL2Hop={onToggle}
        t={t}
      />,
    );

    fireEvent.click(within(getByTestId('l2-hop-0')).getByRole('button'));
    expect(onToggle).toHaveBeenCalledWith(0);
  });

  it('renders L3-only when no L2 path is present', () => {
    const { getByTestId, queryByTestId } = render(
      <PATH_TIMELINE
        result={makeResult({ l2Path: undefined })}
        maxRtt={20_000_000}
        expandedL2Hop={null}
        onToggleL2Hop={vi.fn()}
        t={t}
      />,
    );

    expect(queryByTestId('l2-empty')).toBeNull();
    expect(queryByTestId('l2-hop-0')).toBeNull();
    expect(getByTestId('l3-hop-2')).toBeTruthy();
  });
});
