/**
 * BonjourCard tests (#364).
 *
 * The card's reason to exist is the cross-subnet verdict, so these cover the
 * four states it can report and the two traps behind them: that a browse is
 * never fired without the operator asking, and that "no traffic" is not
 * rendered as "no reflector".
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { BonjourCard } from './BonjourCard';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));
vi.mock('../../api', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));

interface BrowseFixture {
  state: string;
  remoteSubnets?: string[];
  forwardedBy?: string[];
  routedFrom?: string[];
  services?: unknown[];
  durationMs?: number;
}

function browseResult(fixture: BrowseFixture): unknown {
  return {
    interface: 'en0',
    localPrefixes: ['192.168.20.0/24'],
    serviceTypes: ['_airplay._tcp'],
    services: fixture.services ?? [],
    reflector: {
      state: fixture.state,
      remoteSubnets: fixture.remoteSubnets,
      forwardedBy: fixture.forwardedBy,
      routedFrom: fixture.routedFrom,
    },
    responsesObserved: 3,
    truncated: false,
    durationMs: fixture.durationMs ?? 4000,
  };
}

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('BonjourCard', () => {
  it('does not browse until the operator asks', async () => {
    render(<BonjourCard />);

    // A browse puts multicast queries on the segment. Firing one because a
    // page was opened is traffic nobody asked for.
    await waitFor(() => {
      expect(screen.getByTestId('bonjour-browse')).toBeInTheDocument();
    });
    expect(mockGet).not.toHaveBeenCalled();
  });

  it('browses the discovery endpoint when asked', async () => {
    mockGet.mockResolvedValue(browseResult({ state: 'local-only' }));
    render(<BonjourCard />);

    await userEvent.click(screen.getByTestId('bonjour-browse'));

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledWith('/api/v1/discovery/bonjour');
    });
  });

  it('says a reflector is forwarding, naming the subnet and the forwarder', async () => {
    mockGet.mockResolvedValue(
      browseResult({
        state: 'reflected',
        remoteSubnets: ['10.44.40.0/24'],
        forwardedBy: ['192.168.20.1'],
      }),
    );
    render(<BonjourCard />);
    await userEvent.click(screen.getByTestId('bonjour-browse'));

    const verdict = await screen.findByTestId('bonjour-reflector');
    expect(verdict).toHaveAttribute('data-state', 'reflected');
    expect(verdict.textContent).toContain('10.44.40.0/24');
    expect(verdict.textContent).toContain('192.168.20.1');
  });

  it('points at the router, not the reflector, when the source is off-segment', async () => {
    mockGet.mockResolvedValue(browseResult({ state: 'routed', routedFrom: ['10.44.40.1'] }));
    render(<BonjourCard />);
    await userEvent.click(screen.getByTestId('bonjour-browse'));

    const verdict = await screen.findByTestId('bonjour-reflector');
    expect(verdict).toHaveAttribute('data-state', 'routed');
    expect(verdict.textContent).toContain('10.44.40.1');
    expect(verdict.textContent?.toLowerCase()).toContain('router');
  });

  it('does not claim there is no reflector when it simply saw nothing', async () => {
    mockGet.mockResolvedValue(browseResult({ state: 'no-traffic', durationMs: 4000 }));
    render(<BonjourCard />);
    await userEvent.click(screen.getByTestId('bonjour-browse'));

    const verdict = await screen.findByTestId('bonjour-reflector');
    expect(verdict).toHaveAttribute('data-state', 'no-traffic');
    // An idle segment and a blocked one look identical from here, and the
    // sentence has to say so rather than report an absence as a finding.
    expect(verdict.textContent).toContain('4');
    expect(verdict.textContent?.toLowerCase()).toContain('not evidence');
  });

  it('lists each service with its host and port', async () => {
    mockGet.mockResolvedValue(
      browseResult({
        state: 'local-only',
        services: [
          {
            instance: 'Living Room',
            type: '_airplay._tcp',
            host: 'apple-tv.local',
            port: 7000,
            addresses: ['192.168.20.40'],
            origin: 'local',
            sourceAddresses: ['192.168.20.40'],
          },
        ],
      }),
    );
    render(<BonjourCard />);
    await userEvent.click(screen.getByTestId('bonjour-browse'));

    const table = await screen.findByTestId('bonjour-services');
    expect(table.textContent).toContain('Living Room');
    expect(table.textContent).toContain('_airplay._tcp');
    expect(table.textContent).toContain('apple-tv.local');
    expect(table.textContent).toContain('7000');
  });

  it('says a missing SRV record is not port zero', async () => {
    mockGet.mockResolvedValue(
      browseResult({
        state: 'local-only',
        services: [
          {
            instance: 'Studio Display',
            type: '_companion-link._tcp',
            port: 0,
            origin: 'unknown',
          },
        ],
      }),
    );
    render(<BonjourCard />);
    await userEvent.click(screen.getByTestId('bonjour-browse'));

    const table = await screen.findByTestId('bonjour-services');
    expect(table.textContent).not.toContain('0');
    expect(table.textContent?.toLowerCase()).toContain('not advertised');
  });

  it('reports an unfamiliar state as unknown rather than as a raw key', async () => {
    // A server newer than this build may report a state the card has never
    // heard of; rendering "bonjour.reflector.whatever" would be worse than
    // saying so.
    mockGet.mockResolvedValue(browseResult({ state: 'something-new' }));
    render(<BonjourCard />);
    await userEvent.click(screen.getByTestId('bonjour-browse'));

    const verdict = await screen.findByTestId('bonjour-reflector');
    expect(verdict.textContent).not.toContain('bonjour.reflector');
    expect(verdict.textContent?.toLowerCase()).toContain('update seed');
  });

  it('shows the failure when the browse cannot join the group', async () => {
    mockGet.mockRejectedValue(new Error('Could not join the mDNS multicast group'));
    render(<BonjourCard />);
    await userEvent.click(screen.getByTestId('bonjour-browse'));

    const error = await screen.findByTestId('bonjour-error');
    expect(error.textContent).toContain('Could not join the mDNS multicast group');
  });
});
