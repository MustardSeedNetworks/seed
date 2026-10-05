/**
 * PathMonitorCard tests (#165).
 *
 * The monitor runs as a job and its rounds arrive as `pathMonitor` frames, so
 * these drive the card through the job client, the job event stream and the
 * frame subscription: a viewer is never offered the form, each round redraws
 * the hop table, a frame for another destination is ignored, and a stop keeps
 * the final view.
 */

import { act, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PathMonitorUpdate } from '../../hooks/usePathMonitor';
import i18n from '../../i18n';
import type { JobResponse } from '../../types/generated/job-response';
import type { HopStats } from '../../types/generated/path-monitor-update';
import { PathMonitorCard } from './PathMonitorCard';

const role = vi.hoisted(() => ({ canWrite: true }));
vi.mock('../../contexts/RoleContext', () => ({
  useRole: () => ({ canWrite: role.canWrite }),
}));

const jobs = vi.hoisted(() => ({
  submitJob: vi.fn<(req: unknown) => Promise<JobResponse>>(),
  getJob: vi.fn<(id: string) => Promise<JobResponse>>(),
  cancelJob: vi.fn<(id: string) => Promise<JobResponse>>(),
  onJob: (_job: JobResponse): void => undefined,
}));
vi.mock('../../lib/jobsClient', () => ({
  submitJob: (req: unknown) => jobs.submitJob(req),
  getJob: (id: string) => jobs.getJob(id),
  cancelJob: (id: string) => jobs.cancelJob(id),
}));
vi.mock('../../hooks/useJobEvents', () => ({
  useJobEvents: (onJob: (job: JobResponse) => void) => {
    jobs.onJob = onJob;
    return { status: 'open' };
  },
}));

const JOB_ID = 'job-1';
const MS = 1_000_000;

function job(state: string, extra: Partial<JobResponse> = {}): JobResponse {
  return { id: JOB_ID, kind: 'path-monitor', state, progress: 0, ...extra };
}

function hop(ttl: number, overrides: Partial<HopStats> = {}): HopStats {
  return {
    ttl,
    addresses: [{ ip: `10.0.0.${ttl}`, count: 4 }],
    sent: 4,
    received: 4,
    lossPct: 0,
    lastRtt: 2 * MS,
    bestRtt: 1 * MS,
    worstRtt: 3 * MS,
    avgRtt: 2 * MS,
    jitter: 0.5 * MS,
    ...overrides,
  };
}

function snapshot(rounds: number, hops: HopStats[], target = '10.0.0.1'): PathMonitorUpdate {
  return { target, rounds, hops };
}

let frame: (update: PathMonitorUpdate) => void = () => undefined;
const subscribe = vi.fn((handler: (update: PathMonitorUpdate) => void) => {
  frame = handler;
  return () => {
    frame = () => undefined;
  };
});

function renderCard(): ReturnType<typeof render> {
  return render(
    <PathMonitorCard defaultTarget="10.0.0.1" linkDown={false} subscribe={subscribe} />,
  );
}

async function startMonitor(): Promise<void> {
  jobs.submitJob.mockResolvedValue(job('running'));
  jobs.getJob.mockResolvedValue(job('running'));
  await userEvent.click(screen.getByTestId('path-monitor-start'));
  await screen.findByTestId('path-monitor-stop');
}

function hopRow(ttl: number): HTMLElement {
  const row = screen
    .getAllByTestId('path-monitor-hop')
    .find((r) => r.getAttribute('data-ttl') === String(ttl));
  if (!row) {
    throw new Error(`no row for hop ${ttl}`);
  }
  return row;
}

beforeEach(async () => {
  role.canWrite = true;
  jobs.submitJob.mockReset();
  jobs.getJob.mockReset();
  jobs.cancelJob.mockReset();
  jobs.cancelJob.mockResolvedValue(job('running'));
  await i18n.changeLanguage('en');
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('PathMonitorCard', () => {
  it('gives a viewer the reason instead of a start control', () => {
    role.canWrite = false;
    renderCard();

    expect(screen.queryByTestId('path-monitor-start')).not.toBeInTheDocument();
    expect(screen.getByTestId('path-monitor-read-only')).toHaveTextContent(/operator role/);
    expect(jobs.submitJob).not.toHaveBeenCalled();
  });

  it('starts a monitor on the gateway without asking for knobs', async () => {
    renderCard();
    await startMonitor();

    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'path-monitor',
      params: { destination: '10.0.0.1' },
    });
    expect(screen.getByTestId('path-monitor')).toHaveAttribute('data-phase', 'running');
    expect(screen.getByTestId('path-monitor-target')).toBeDisabled();
  });

  it('redraws the hop table from each round and ignores other destinations', async () => {
    renderCard();
    await startMonitor();

    act(() => frame(snapshot(1, [hop(1)])));
    expect(screen.getByTestId('path-monitor-rounds')).toHaveTextContent('1 round to 10.0.0.1');
    expect(screen.getAllByTestId('path-monitor-hop')).toHaveLength(1);

    act(() => frame(snapshot(9, [hop(1), hop(2)], '192.0.2.9')));
    expect(screen.getByTestId('path-monitor-rounds')).toHaveTextContent('1 round to 10.0.0.1');

    act(() =>
      frame(
        snapshot(4, [
          hop(1),
          hop(2, {
            sent: 4,
            received: 3,
            lossPct: 25,
            addresses: [
              { ip: '10.0.0.2', hostname: 'core-a', count: 2 },
              { ip: '10.0.0.3', count: 1 },
            ],
          }),
        ]),
      ),
    );
    expect(screen.getByTestId('path-monitor-rounds')).toHaveTextContent('4 rounds to 10.0.0.1');
    const lossy = hopRow(2);
    expect(within(lossy).getByTestId('path-monitor-hop-loss')).toHaveTextContent('25%');
    // Two addresses answering one TTL is a load-balanced path, not one hop.
    expect(within(lossy).getByTestId('path-monitor-hop-address')).toHaveTextContent('core-a +1');
    expect(within(lossy).getByTestId('path-monitor-hop-address')).toHaveAttribute(
      'title',
      'core-a (10.0.0.2), 10.0.0.3',
    );
    expect(within(hopRow(1)).getByTestId('path-monitor-hop-avg')).toHaveTextContent('2.0ms');
  });

  it('shows a hop that never answered as silent, not as zero latency', async () => {
    renderCard();
    await startMonitor();

    act(() =>
      frame(
        snapshot(3, [
          hop(1),
          hop(2, {
            addresses: [],
            received: 0,
            lossPct: 100,
            lastRtt: 0,
            bestRtt: 0,
            worstRtt: 0,
            avgRtt: 0,
            jitter: 0,
          }),
        ]),
      ),
    );
    const silent = hopRow(2);
    expect(within(silent).getByTestId('path-monitor-hop-address')).toHaveTextContent('*');
    expect(within(silent).getByTestId('path-monitor-hop-loss')).toHaveTextContent('100%');
    expect(within(silent).getByTestId('path-monitor-hop-avg')).toHaveTextContent('---');
  });

  it('keeps the final view when stopped', async () => {
    renderCard();
    await startMonitor();
    act(() => frame(snapshot(2, [hop(1)])));

    jobs.cancelJob.mockResolvedValue(job('running'));
    await userEvent.click(screen.getByTestId('path-monitor-stop'));
    expect(jobs.cancelJob).toHaveBeenCalledWith(JOB_ID);
    expect(screen.getByTestId('path-monitor')).toHaveAttribute('data-phase', 'stopping');
    // The table stays up while the stop is in flight.
    expect(screen.getAllByTestId('path-monitor-hop')).toHaveLength(1);

    act(() => jobs.onJob(job('succeeded', { result: snapshot(3, [hop(1), hop(2)]) })));
    expect(screen.getByTestId('path-monitor')).toHaveAttribute('data-phase', 'stopped');
    expect(screen.getByTestId('path-monitor-rounds')).toHaveTextContent('3 rounds to 10.0.0.1');
    expect(screen.getAllByTestId('path-monitor-hop')).toHaveLength(2);
    expect(screen.getByTestId('path-monitor-start')).toBeEnabled();
  });

  it('reports a refused start with the reason', async () => {
    renderCard();
    jobs.submitJob.mockRejectedValue(new Error('job runner at capacity'));
    await userEvent.click(screen.getByTestId('path-monitor-start'));

    expect(await screen.findByTestId('path-monitor-failed')).toBeInTheDocument();
    expect(screen.getByTestId('path-monitor-error-detail')).toHaveTextContent(
      'job runner at capacity',
    );
    expect(screen.getByTestId('path-monitor-start')).toBeEnabled();
  });

  it('lets the operator retry a stop that did not reach the server', async () => {
    renderCard();
    await startMonitor();

    jobs.cancelJob.mockRejectedValue(new Error('network'));
    await userEvent.click(screen.getByTestId('path-monitor-stop'));

    expect(await screen.findByText('The monitor did not stop. Try again.')).toBeInTheDocument();
    expect(screen.getByTestId('path-monitor-stop')).toBeEnabled();
  });

  it('stops a running monitor when the link goes down, and says so', async () => {
    const { rerender } = renderCard();
    await startMonitor();
    act(() => frame(snapshot(2, [hop(1)])));

    rerender(<PathMonitorCard defaultTarget="10.0.0.1" linkDown subscribe={subscribe} />);
    expect(jobs.cancelJob).toHaveBeenCalledWith(JOB_ID);
    act(() => jobs.onJob(job('succeeded', { result: snapshot(2, [hop(1)]) })));

    expect(screen.getByTestId('path-monitor')).toHaveAttribute('data-phase', 'stopped');
    expect(screen.getByTestId('path-monitor-link-down')).toHaveTextContent(
      'Stopped because the link went down.',
    );
    expect(screen.getAllByTestId('path-monitor-hop')).toHaveLength(1);
  });

  it('runs on a link that was already down when it started', async () => {
    render(<PathMonitorCard defaultTarget="10.0.0.1" linkDown subscribe={subscribe} />);
    await startMonitor();
    act(() => frame(snapshot(2, [hop(1)])));

    expect(screen.getByTestId('path-monitor')).toHaveAttribute('data-phase', 'running');
    expect(jobs.cancelJob).not.toHaveBeenCalled();
  });

  it('stops a running monitor when the card unmounts', async () => {
    jobs.submitJob.mockResolvedValue(job('running'));
    jobs.getJob.mockResolvedValue(job('running'));
    jobs.cancelJob.mockResolvedValue(job('running'));
    const { unmount } = renderCard();
    await userEvent.click(screen.getByTestId('path-monitor-start'));
    await screen.findByTestId('path-monitor-stop');

    unmount();
    expect(jobs.cancelJob).toHaveBeenCalledWith(JOB_ID);
  });
});
