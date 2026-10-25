/**
 * IP SLA (UI-SEED-48): operations keep the server's order with each latest
 * run worded and stated, the detail charts the selected operation's trend
 * and re-reads on a new window, and empty or failed reads say so.
 */
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { IPSLAHistoryResponse } from '../types/generated/ip-sla-history-response';
import type { IPSLAOperationResponse } from '../types/generated/ip-sla-operations-response';

const OBSERVED = '2026-10-21T12:00:00Z';

/** The server lists failing operations first. */
const timedOut: IPSLAOperationResponse = {
  targetId: 't1',
  targetName: 'edge-rtr',
  observedAt: OBSERVED,
  index: 20,
  tag: 'to-dc2',
  rttType: 1,
  target: '10.9.0.1',
  thresholdMs: 100,
  frequencySeconds: 60,
  status: 1,
  latest: { completed: false, sense: 4, rttMs: 0 },
};

const voice: IPSLAOperationResponse = {
  targetId: 't1',
  targetName: 'edge-rtr',
  observedAt: OBSERVED,
  index: 10,
  rttType: 9,
  target: '10.9.0.5',
  thresholdMs: 50,
  frequencySeconds: 30,
  status: 1,
  latest: {
    completed: true,
    sense: 3,
    rttMs: 72,
    jitter: { avgJitterMs: 4, received: 98, lost: 2, mos: 4.21 },
  },
};

const idle: IPSLAOperationResponse = {
  targetId: 't2',
  targetName: 'core-sw',
  observedAt: OBSERVED,
  index: 5,
  rttType: 99,
  thresholdMs: 5000,
  frequencySeconds: 1,
  status: 2,
};

function trend(points: IPSLAHistoryResponse['points']): IPSLAHistoryResponse {
  return {
    targetId: 't1',
    index: 20,
    range: '24h',
    from: '2026-10-20T12:00:00Z',
    to: '2026-10-21T12:00:00Z',
    bucketSeconds: 360,
    points,
  };
}

let operations: IPSLAOperationResponse[] = [];
let listError: Error | null = null;
let history: IPSLAHistoryResponse = trend([]);
const reads: string[] = [];

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    api: {
      get: async (path: string): Promise<unknown> => {
        reads.push(path);
        if (path.startsWith('/api/v1/ip-sla/operations/history')) {
          return history;
        }
        if (listError) {
          throw listError;
        }
        return { operations, count: operations.length };
      },
    },
  };
});

const { IpSlaPage } = await import('./IpSlaPage');

describe('IpSlaPage', () => {
  beforeEach(() => {
    operations = [timedOut, voice, idle];
    listError = null;
    history = trend([]);
    reads.length = 0;
  });

  it('words each latest run and keeps the failing operation first', async () => {
    render(<IpSlaPage />);

    const failed = await screen.findByTestId('ipsla-op-t1#20');
    expect(failed).toHaveTextContent('to-dc2');
    expect(failed).toHaveTextContent('edge-rtr → 10.9.0.1');
    expect(failed).toHaveTextContent('Timed out');
    expect(failed).toHaveAttribute('data-state', 'crit');

    const slow = screen.getByTestId('ipsla-op-t1#10');
    // An operation without a tag is named by its index.
    expect(slow).toHaveTextContent('Operation 10');
    expect(slow).toHaveTextContent('72 ms, over threshold');
    expect(slow).toHaveAttribute('data-state', 'warn');

    const never = screen.getByTestId('ipsla-op-t2#5');
    expect(never).toHaveTextContent('No run yet');
    expect(never).toHaveAttribute('data-state', 'unknown');

    expect(screen.getByTestId('ipsla-counts')).toHaveTextContent('3 operations · 1 failing');
  });

  it('opens the first operation and reads its trend for the last 24 hours', async () => {
    render(<IpSlaPage />);

    expect(await screen.findByTestId('ipsla-type')).toHaveTextContent('ICMP echo');
    expect(screen.getByTestId('ipsla-threshold')).toHaveTextContent('100 ms');
    // An echo operation has no jitter facts.
    expect(screen.queryByTestId('ipsla-jitter')).not.toBeInTheDocument();
    expect(await screen.findByTestId('ipsla-history-empty')).toHaveTextContent(
      'No run in this window.',
    );
    expect(reads).toContain('/api/v1/ip-sla/operations/history?target=t1&index=20&range=24h');
  });

  it('charts a jitter operation with jitter and loss, and re-reads on a new window', async () => {
    history = trend([
      {
        sampledAt: '2026-10-21T11:00:00Z',
        runs: 12,
        completedPct: 100,
        rttMs: 40,
        jitterMs: 3,
        lossPct: 0,
      },
      {
        sampledAt: '2026-10-21T11:06:00Z',
        runs: 12,
        completedPct: 75,
        rttMs: 72.4,
        jitterMs: 6,
        lossPct: 2,
      },
    ]);
    const user = userEvent.setup();
    render(<IpSlaPage />);

    await user.click(await screen.findByTestId('ipsla-op-t1#10'));
    expect(screen.getByTestId('ipsla-type')).toHaveTextContent('UDP jitter');
    expect(screen.getByTestId('ipsla-jitter')).toHaveTextContent('4 ms');
    expect(screen.getByText('2 of 100')).toBeInTheDocument();
    expect(screen.getByText('4.21')).toBeInTheDocument();

    expect(await screen.findByTestId('ipsla-history-timing-peaks')).toHaveTextContent(
      'Peak RTT 72.4 ms · Peak Jitter 6 ms',
    );
    expect(screen.getByTestId('ipsla-history-outcome-peaks')).toHaveTextContent(
      'Peak Completed 100% · Peak Loss 2%',
    );
    expect(
      within(screen.getByTestId('ipsla-history-timing')).getByRole('img'),
    ).toHaveAccessibleName(/^Round trip: Peak RTT 72.4 ms/);

    await user.selectOptions(screen.getByTestId('ipsla-history-range'), '7d');
    expect(reads.at(-1)).toBe('/api/v1/ip-sla/operations/history?target=t1&index=10&range=7d');
  });

  it('names an unknown type by number and an operation that is not running', async () => {
    const user = userEvent.setup();
    render(<IpSlaPage />);

    await user.click(await screen.findByTestId('ipsla-op-t2#5'));
    expect(screen.getByTestId('ipsla-type')).toHaveTextContent('Type 99');
    expect(screen.getByText('Not running')).toBeInTheDocument();
    expect(screen.getByText('Every second')).toBeInTheDocument();
  });

  it('says so when no operation is polled yet', async () => {
    operations = [];
    render(<IpSlaPage />);

    expect(await screen.findByText(/No IP SLA operation yet/)).toBeInTheDocument();
    expect(screen.getByTestId('ipsla-counts')).toHaveTextContent('0 operations');
    expect(reads.some((path) => path.includes('/history'))).toBe(false);
  });

  it('shows a failed read as an alert', async () => {
    listError = new Error('store unavailable');
    render(<IpSlaPage />);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not load IP SLA operations: store unavailable',
    );
  });
});
