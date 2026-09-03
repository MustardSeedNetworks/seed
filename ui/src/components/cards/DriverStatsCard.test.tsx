/**
 * DriverStatsCard tests (#416).
 *
 * The counters are only useful with their meaning attached — "rx_crc_errors: 4"
 * tells an operator nothing they can act on, "frames arrived corrupted, usually
 * cabling" tells them where to go. So the assertions are on the explanation and
 * on the platform gate, which is what stops macOS showing an empty table that
 * reads as "no errors".
 */

import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { DriverStatsCard } from './DriverStatsCard';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));
vi.mock('../../api', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));

const supported = {
  capability: 'driver_statistics',
  title: 'Driver error counters',
  level: 'full',
};
const unsupported = {
  capability: 'driver_statistics',
  title: 'Driver error counters',
  level: 'none',
  note: 'ethtool is a Linux ioctl interface; macOS has no equivalent, so driver error counters cannot be read.',
};

function respond(capability: unknown, stats: unknown): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/status')) {
      return Promise.resolve({ capabilities: [capability] });
    }

    return Promise.resolve(stats);
  });
}

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('DriverStatsCard', () => {
  it('shows each counter with its meaning and the driver’s own name for it', async () => {
    respond(supported, {
      interface: 'eth0',
      total: 84,
      counters: [
        {
          key: 'rx_crc_errors',
          label: 'CRC errors',
          value: 4,
          meaning:
            'Frames arrived corrupted. Usually cabling, a bad port, or a duplex mismatch — not congestion.',
        },
      ],
    });
    render(<DriverStatsCard />);

    expect(await screen.findByText('CRC errors')).toBeInTheDocument();
    expect(screen.getByText('4')).toBeInTheDocument();
    expect(
      screen.getByText(/Usually cabling, a bad port, or a duplex mismatch/),
    ).toBeInTheDocument();
    // The raw key so an operator can correlate with `ethtool -S`.
    expect(screen.getByText('rx_crc_errors')).toBeInTheDocument();
  });

  it('says the curated set is a selection, not the whole story', async () => {
    respond(supported, {
      interface: 'eth0',
      total: 84,
      counters: [
        { key: 'collisions', label: 'TX collisions', value: 0, meaning: 'Should be zero.' },
      ],
    });
    render(<DriverStatsCard />);

    expect(
      await screen.findByText('Showing 1 of 84 counters the driver reports.'),
    ).toBeInTheDocument();
  });

  it('explains the platform instead of showing an empty table', async () => {
    respond(unsupported, {});
    render(<DriverStatsCard />);

    expect(await screen.findByTestId('feature-unavailable')).toBeInTheDocument();
    expect(screen.getByText(/macOS has no equivalent/)).toBeInTheDocument();
    // An empty counter table would read as "this link is clean", which is the
    // most reassuring possible wrong answer.
    expect(screen.queryByText('Driver Error Counters')).not.toBeInTheDocument();
  });

  it('does not claim a clean link when the driver exposes no curated counters', async () => {
    respond(supported, { interface: 'eth0', total: 12, counters: [] });
    render(<DriverStatsCard />);

    expect(
      await screen.findByText('This driver exposes none of the counters worth watching.'),
    ).toBeInTheDocument();
  });

  it('surfaces a read failure', async () => {
    mockGet.mockImplementation((path: string) =>
      path.includes('/status')
        ? Promise.resolve({ capabilities: [supported] })
        : Promise.reject(new Error('operation not permitted')),
    );
    render(<DriverStatsCard />);

    await waitFor(() => {
      expect(screen.getByText('operation not permitted')).toBeInTheDocument();
    });
  });
});
