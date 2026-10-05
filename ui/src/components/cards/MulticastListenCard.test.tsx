/**
 * MulticastListenCard tests (#399).
 *
 * The listen runs as a job, so these drive the card through the job client
 * and the job event stream: a viewer is never offered the form, a stop ends
 * in what was heard, and a silent group reads as the finding it is rather
 * than as a failed run.
 */

import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import type { JobResponse } from '../../types/generated/job-response';
import type { ListenResult } from '../../types/generated/multicast-listen-response';
import { MulticastListenCard } from './MulticastListenCard';

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

function job(state: string, extra: Partial<JobResponse> = {}): JobResponse {
  return { id: JOB_ID, kind: 'multicast-listen', state, progress: 0, ...extra };
}

function listenResult(overrides: Partial<ListenResult> = {}): ListenResult {
  return {
    group: '239.1.1.1',
    port: 5000,
    interface: 'eth0',
    listenedMs: 10000,
    packets: 1200,
    bytes: 1504800,
    packetsPerSecond: 120,
    sources: [
      { address: '192.0.2.20', packets: 1150, bytes: 1442100 },
      { address: '192.0.2.21', packets: 50, bytes: 62700 },
    ],
    sourcesTruncated: false,
    groupFiltered: true,
    ...overrides,
  };
}

function emit(event: JobResponse): void {
  act(() => jobs.onJob(event));
}

async function fillAndStart(): Promise<void> {
  await userEvent.type(screen.getByTestId('multicast-listen-group'), '239.1.1.1');
  await userEvent.type(screen.getByTestId('multicast-listen-port'), '5000');
  jobs.submitJob.mockResolvedValue(job('running'));
  jobs.getJob.mockResolvedValue(job('running'));
  await userEvent.click(screen.getByTestId('multicast-listen-start'));
  await screen.findByTestId('multicast-listen-stop');
}

beforeEach(async () => {
  role.canWrite = true;
  jobs.submitJob.mockReset();
  jobs.getJob.mockReset();
  jobs.cancelJob.mockReset();
  await i18n.changeLanguage('en');
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('MulticastListenCard', () => {
  it('gives a viewer the reason instead of a start control', () => {
    role.canWrite = false;
    render(<MulticastListenCard defaultInterface="eth0" />);

    expect(screen.queryByTestId('multicast-listen-start')).not.toBeInTheDocument();
    expect(screen.queryByTestId('multicast-listen-group')).not.toBeInTheDocument();
    expect(screen.getByTestId('multicast-listen-read-only')).toHaveTextContent(/operator role/);
    expect(jobs.submitJob).not.toHaveBeenCalled();
  });

  it('cannot start until a group and a port are given', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    const start = screen.getByTestId('multicast-listen-start');

    expect(start).toBeDisabled();
    await userEvent.type(screen.getByTestId('multicast-listen-group'), '239.1.1.1');
    expect(start).toBeDisabled();
    await userEvent.type(screen.getByTestId('multicast-listen-port'), '70000');
    expect(start).toBeDisabled();
    expect(screen.getByText('Enter a whole number from 1 to 65535.')).toBeInTheDocument();

    const port = screen.getByTestId('multicast-listen-port');
    await userEvent.clear(port);
    await userEvent.type(port, '5000');
    expect(start).toBeEnabled();
  });

  it('refuses a duration past the minute the job allows', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    await userEvent.type(screen.getByTestId('multicast-listen-group'), '239.1.1.1');
    await userEvent.type(screen.getByTestId('multicast-listen-port'), '5000');

    const duration = screen.getByTestId('multicast-listen-duration');
    await userEvent.clear(duration);
    await userEvent.type(duration, '61');

    expect(screen.getByTestId('multicast-listen-start')).toBeDisabled();
    expect(screen.getByText('Enter a whole number from 1 to 60.')).toBeInTheDocument();
  });

  it('listens on the current interface for ten seconds unless told otherwise', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    await fillAndStart();

    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'multicast-listen',
      params: { group: '239.1.1.1', port: 5000, interface: 'eth0', durationSeconds: 10 },
    });
    expect(screen.getByTestId('multicast-listen')).toHaveAttribute('data-phase', 'running');
    expect(screen.getByTestId('multicast-listen-running')).toHaveTextContent(
      'Listening for 239.1.1.1 port 5000 on eth0.',
    );
  });

  it('stops through the job and shows what was heard, busiest sender first', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    await fillAndStart();

    jobs.cancelJob.mockResolvedValue(job('running'));
    await userEvent.click(screen.getByTestId('multicast-listen-stop'));
    expect(jobs.cancelJob).toHaveBeenCalledWith(JOB_ID);
    expect(screen.getByTestId('multicast-listen')).toHaveAttribute('data-phase', 'stopping');

    // A stopped listen keeps what it heard: the job ends succeeded.
    emit(job('succeeded', { result: listenResult() }));

    expect(screen.getByTestId('multicast-listen-totals')).toHaveTextContent(
      '1200 packets, 1.4 MB in 10 s (120.0/s) for 239.1.1.1 port 5000 on eth0',
    );
    const rows = within(screen.getByTestId('multicast-listen-sources')).getAllByRole('row');
    expect(rows[1]).toHaveTextContent('192.0.2.20');
    expect(rows[2]).toHaveTextContent('192.0.2.21');
    expect(screen.queryByTestId('multicast-listen-silent')).not.toBeInTheDocument();
    expect(screen.queryByTestId('multicast-listen-unfiltered')).not.toBeInTheDocument();
    expect(screen.getByTestId('multicast-listen-start')).toBeEnabled();
  });

  it('says a silent group points at the sender or the path, not that the run failed', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    await fillAndStart();

    emit(
      job('succeeded', {
        result: listenResult({ packets: 0, bytes: 0, packetsPerSecond: 0, sources: [] }),
      }),
    );

    expect(screen.getByTestId('multicast-listen-silent')).toHaveTextContent(/IGMP or MLD snooping/);
    expect(screen.queryByTestId('multicast-listen-sources')).not.toBeInTheDocument();
    expect(screen.queryByTestId('multicast-listen-failed')).not.toBeInTheDocument();
    expect(screen.getByTestId('multicast-listen')).toHaveAttribute('data-phase', 'finished');
  });

  it('notes a truncated sender list and counts the platform could not filter', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    await fillAndStart();

    emit(
      job('succeeded', { result: listenResult({ sourcesTruncated: true, groupFiltered: false }) }),
    );

    expect(screen.getByTestId('multicast-listen-truncated')).toBeInTheDocument();
    expect(screen.getByTestId('multicast-listen-unfiltered')).toBeInTheDocument();
  });

  it('shows a failed listen with the reason the daemon gave', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    await fillAndStart();

    emit(
      job('failed', {
        error: 'interface cannot join a multicast group: lo does not support multicast',
      }),
    );

    expect(screen.getByRole('alert')).toHaveTextContent('The listen did not run.');
    expect(screen.getByTestId('multicast-listen-error-detail')).toHaveTextContent(
      'lo does not support multicast',
    );
    expect(screen.queryByTestId('multicast-listen-result')).not.toBeInTheDocument();
  });

  it('says so when the start request itself is refused', async () => {
    render(<MulticastListenCard defaultInterface="eth0" />);
    await userEvent.type(screen.getByTestId('multicast-listen-group'), '239.1.1.1');
    await userEvent.type(screen.getByTestId('multicast-listen-port'), '5000');
    jobs.submitJob.mockRejectedValue(new Error('Forbidden'));

    await userEvent.click(screen.getByTestId('multicast-listen-start'));

    await waitFor(() => {
      expect(screen.getByTestId('multicast-listen')).toHaveAttribute('data-phase', 'failed');
    });
    expect(screen.getByTestId('multicast-listen-error-detail')).toHaveTextContent('Forbidden');
  });

  it('renders Spanish under es', async () => {
    await i18n.changeLanguage('es');
    render(<MulticastListenCard defaultInterface="eth0" />);
    await fillAndStart();

    emit(
      job('succeeded', {
        result: listenResult({ packets: 0, bytes: 0, packetsPerSecond: 0, sources: [] }),
      }),
    );

    expect(screen.getByTestId('multicast-listen-totals')).toHaveTextContent(
      '0 paquetes, 0 B en 10 s (0.0/s) para 239.1.1.1 puerto 5000 en eth0',
    );
    expect(screen.getByTestId('multicast-listen-silent')).toHaveTextContent(/No llegó nada/);
  });
});
