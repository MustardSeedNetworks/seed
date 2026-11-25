/**
 * NeighbourCacheCard tests (#328).
 *
 * The reader has existed cross-platform for a long time; the entries were
 * folded into device discovery and never surfaced. These cover the exposure —
 * that the card asks the local endpoint rather than the topology one, and that
 * an IPv4 and an IPv6 entry are distinguishable.
 */

import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { NeighbourCacheCard } from './NeighbourCacheCard';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));
vi.mock('../../api', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('NeighbourCacheCard', () => {
  it('reads the local cache, not the topology ARP table', async () => {
    mockGet.mockResolvedValue({ entries: [], total: 0 });
    render(<NeighbourCacheCard />);

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledWith('/api/v1/network/neighbours');
    });
    // /topology/arp answers a different question — what a remote switch
    // reports over SNMP — and confusing the two is the trap #328 names.
    expect(mockGet).not.toHaveBeenCalledWith(expect.stringContaining('/topology/arp'));
  });

  it('shows each entry with its vendor, interface and state', async () => {
    mockGet.mockResolvedValue({
      entries: [
        {
          ip: '192.0.2.1',
          mac: '74:ac:b9:3b:af:40',
          vendor: 'Ubiquiti',
          interface: 'en0',
          state: 'REACHABLE',
          family: 'ipv4',
        },
      ],
      total: 1,
    });
    render(<NeighbourCacheCard />);

    expect(await screen.findByText('192.0.2.1')).toBeInTheDocument();
    expect(screen.getByText('74:ac:b9:3b:af:40')).toBeInTheDocument();
    expect(screen.getByText('Ubiquiti')).toBeInTheDocument();
    expect(screen.getByText('en0')).toBeInTheDocument();
    expect(screen.getByText('REACHABLE')).toBeInTheDocument();
  });

  it('distinguishes IPv6 entries from IPv4 ones', async () => {
    mockGet.mockResolvedValue({
      entries: [
        { ip: '192.0.2.1', mac: 'aa:bb:cc:dd:ee:01', family: 'ipv4' },
        { ip: '2001:db8::1', mac: 'aa:bb:cc:dd:ee:02', family: 'ipv6' },
      ],
      total: 2,
    });
    render(<NeighbourCacheCard />);

    expect(await screen.findByText('IPv4')).toBeInTheDocument();
    expect(screen.getByText('IPv6')).toBeInTheDocument();
  });

  it('says the cache is empty rather than rendering a headerless table', async () => {
    mockGet.mockResolvedValue({ entries: [], total: 0 });
    render(<NeighbourCacheCard />);

    expect(
      await screen.findByText(
        'The neighbour cache is empty. Nothing on this link has been resolved yet.',
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('surfaces a read failure instead of showing an empty cache', async () => {
    mockGet.mockRejectedValue(new Error('permission denied'));
    render(<NeighbourCacheCard />);

    expect(await screen.findByText('permission denied')).toBeInTheDocument();
  });

  it('filters the entries from the search box and says when none match', async () => {
    mockGet.mockResolvedValue({
      entries: [
        { ip: '192.0.2.1', mac: 'aa:bb:cc:dd:ee:01', vendor: 'Ubiquiti', family: 'ipv4' },
        { ip: '192.0.2.2', mac: 'aa:bb:cc:dd:ee:02', vendor: 'Cisco', family: 'ipv4' },
      ],
      total: 2,
    });
    render(<NeighbourCacheCard />);
    await screen.findByText('192.0.2.1');

    const search = screen.getByRole('searchbox', { name: 'Filter rows' });
    fireEvent.change(search, { target: { value: 'cisco' } });
    expect(screen.queryByText('192.0.2.1')).not.toBeInTheDocument();
    expect(screen.getByText('192.0.2.2')).toBeInTheDocument();

    fireEvent.change(search, { target: { value: 'nothing-like-this' } });
    expect(screen.getByText('No rows match the filter')).toBeInTheDocument();
  });

  it('sorts by a column header', async () => {
    mockGet.mockResolvedValue({
      entries: [
        { ip: '192.0.2.9', mac: 'aa:bb:cc:dd:ee:09', family: 'ipv4' },
        { ip: '192.0.2.1', mac: 'aa:bb:cc:dd:ee:01', family: 'ipv4' },
      ],
      total: 2,
    });
    render(<NeighbourCacheCard />);
    await screen.findByText('192.0.2.9');

    // Sorted by address ascending to begin with.
    const body = (): HTMLElement => screen.getAllByRole('rowgroup')[1] as HTMLElement;
    expect(within(body()).getAllByRole('row')[0]).toHaveTextContent('192.0.2.1');
    fireEvent.click(screen.getByRole('button', { name: /Address/ }));
    expect(within(body()).getAllByRole('row')[0]).toHaveTextContent('192.0.2.9');
  });
});
