/**
 * InterfaceHistory (UI-SEED-21): loading, error and an empty window are
 * distinct from a drawn history; the window control re-reads; a stretch with
 * no poll breaks the line.
 */
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { HistoryRange, UseInterfaceHistoryResult } from '../../hooks/useInterfaceHistory';
import i18n from '../../i18n';
import type {
  InterfaceHistoryResponse,
  InterfaceRatesResponse,
} from '../../types/generated/interface-history-response';
import type { InterfaceStatsResponse } from '../../types/generated/interface-stats-list-response';

const IFACE: InterfaceStatsResponse = {
  targetId: 't1',
  targetName: 'core-sw',
  ifIndex: 3,
  name: 'Gi0/3',
  operStatus: 'up',
  speedBps: 1_000_000_000,
};

function point(
  sampledAt: string,
  extra: Partial<InterfaceRatesResponse> = {},
): InterfaceRatesResponse {
  return {
    sampledAt,
    inErrorsPerSec: 0,
    outErrorsPerSec: 0,
    inDiscardsPerSec: 0,
    outDiscardsPerSec: 0,
    ...extra,
  };
}

function history(points: InterfaceRatesResponse[]): InterfaceHistoryResponse {
  return {
    targetId: 't1',
    ifIndex: 3,
    range: '24h',
    from: '2026-10-07T12:00:00Z',
    to: '2026-10-08T12:00:00Z',
    bucketSeconds: 360,
    points,
  };
}

const state: { current: UseInterfaceHistoryResult; calls: [string, number, HistoryRange][] } = {
  current: { history: null, loading: true, error: null },
  calls: [],
};

vi.mock('../../hooks/useInterfaceHistory', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../hooks/useInterfaceHistory')>()),
  useInterfaceHistory: (
    target: string,
    ifIndex: number,
    range: HistoryRange,
  ): UseInterfaceHistoryResult => {
    state.calls.push([target, ifIndex, range]);
    return state.current;
  },
}));

const { InterfaceHistory, seriesPath } = await import('./InterfaceHistory');

beforeEach(async () => {
  await i18n.changeLanguage('en');
});

afterEach(() => {
  state.current = { history: null, loading: true, error: null };
  state.calls = [];
});

describe('InterfaceHistory states', () => {
  it('shows loading, then focuses its heading', () => {
    render(<InterfaceHistory iface={IFACE} onClose={vi.fn()} />);
    expect(screen.getByTestId('interface-history-loading')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Gi0/3' })).toHaveFocus();
    expect(state.calls.at(-1)).toEqual(['t1', 3, '24h']);
  });

  it('says so when the window holds no rated poll', () => {
    state.current = { history: history([]), loading: false, error: null };
    render(<InterfaceHistory iface={IFACE} onClose={vi.fn()} />);
    expect(screen.getByTestId('interface-history-empty')).toBeInTheDocument();
    expect(screen.queryByTestId('interface-history-traffic')).not.toBeInTheDocument();
  });

  it('shows a failed read as an alert', () => {
    state.current = { history: null, loading: false, error: 'boom' };
    render(<InterfaceHistory iface={IFACE} onClose={vi.fn()} />);
    expect(screen.getByRole('alert')).toHaveTextContent('boom');
  });

  it('draws traffic and errors with their peaks', () => {
    state.current = {
      history: history([
        point('2026-10-08T11:00:00Z', { inOctetsPerSec: 1_250_000, outOctetsPerSec: 125_000 }),
        point('2026-10-08T11:06:00Z', {
          inOctetsPerSec: 250_000,
          inErrorsPerSec: 2,
          outErrorsPerSec: 1,
        }),
      ]),
      loading: false,
      error: null,
    };
    render(<InterfaceHistory iface={IFACE} onClose={vi.fn()} />);
    expect(screen.getByTestId('interface-history-traffic-peaks')).toHaveTextContent(
      'Peak In 10 Mbps · Peak Out 1 Mbps',
    );
    expect(screen.getByTestId('interface-history-errors-peaks')).toHaveTextContent(
      'Peak Errors 3/s · Peak Discards 0/s',
    );
    expect(
      within(screen.getByTestId('interface-history-traffic')).getByRole('img'),
    ).toHaveAccessibleName(/^Traffic: Peak In 10 Mbps/);
  });

  it('re-reads on a new window and closes', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<InterfaceHistory iface={IFACE} onClose={onClose} />);
    await user.selectOptions(screen.getByTestId('interface-history-range'), '7d');
    expect(state.calls.at(-1)).toEqual(['t1', 3, '7d']);
    await user.click(screen.getByTestId('interface-history-close'));
    expect(onClose).toHaveBeenCalledOnce();
  });
});

describe('seriesPath', () => {
  const from = Date.parse('2026-10-08T00:00:00Z');
  const to = from + 600_000;
  const at = (s: number): string => new Date(from + s * 1000).toISOString();
  const value = (p: InterfaceRatesResponse): number | undefined => p.inOctetsPerSec;

  it('joins polls within the gap and scales to the peak', () => {
    const d = seriesPath(
      [point(at(0), { inOctetsPerSec: 0 }), point(at(60), { inOctetsPerSec: 10 })],
      value,
      from,
      to,
      10,
      180_000,
    );
    expect(d).toBe('M0.0 120.0 L60.0 0.0');
  });

  it('breaks the line across a gap and a missing value', () => {
    const d = seriesPath(
      [
        point(at(0), { inOctetsPerSec: 5 }),
        point(at(300), { inOctetsPerSec: 5 }),
        point(at(360)),
        point(at(420), { inOctetsPerSec: 5 }),
      ],
      value,
      from,
      to,
      10,
      180_000,
    );
    expect(d).toBe('M0.0 60.0 M300.0 60.0 M420.0 60.0');
  });
});
