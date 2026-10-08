/**
 * PathMTUCard tests (#435).
 *
 * The check runs as a `path-mtu` job. These drive it through the job client
 * and event stream and pin how each status reads: a measured MTU equal to the
 * local link's is healthy, a smaller one names the cause, a silent drop is a
 * lower bound, and an unreachable destination shows no size at all.
 */

import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { PMTUDResult } from '../../hooks/usePathChecks';
import i18n from '../../i18n';
import type { JobResponse } from '../../types/generated/job-response';
import { PathMTUCard } from './PathMTUCard';

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
  return { id: JOB_ID, kind: 'path-mtu', state, progress: 0, ...extra };
}

function result(overrides: Partial<PMTUDResult>): PMTUDResult {
  return {
    target: '198.51.100.7',
    targetIp: '198.51.100.7',
    status: 'ok',
    pathMtu: 1500,
    localMtu: 1500,
    probes: 11,
    ...overrides,
  };
}

async function measure(answer: PMTUDResult): Promise<void> {
  jobs.submitJob.mockResolvedValue(job('running'));
  jobs.getJob.mockResolvedValue(job('running'));
  render(<PathMTUCard />);
  await userEvent.type(screen.getByTestId('path-mtu-target'), '198.51.100.7');
  await userEvent.click(screen.getByTestId('path-mtu-start'));
  await screen.findByTestId('path-mtu-stop');
  act(() => jobs.onJob(job('succeeded', { result: answer })));
}

beforeEach(async () => {
  role.canWrite = true;
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('PathMTUCard', () => {
  it('gives a viewer the reason instead of a start control', () => {
    role.canWrite = false;
    render(<PathMTUCard />);

    expect(screen.queryByTestId('path-mtu-start')).not.toBeInTheDocument();
    expect(screen.getByTestId('path-mtu-read-only')).toHaveTextContent(/operator role/);
  });

  it('submits the destination as a path-mtu job', async () => {
    await measure(result({}));

    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'path-mtu',
      params: { destination: '198.51.100.7' },
    });
  });

  it('reads a path that carries the local MTU as healthy', async () => {
    await measure(result({}));

    expect(screen.getByTestId('path-mtu-value')).toHaveTextContent('1500');
    expect(screen.getByTestId('path-mtu-verdict')).toHaveTextContent(
      'The whole path carries the local link’s 1500 bytes.',
    );
    expect(screen.getByTestId('path-mtu-probes')).toHaveTextContent(
      '11 probes sent to 198.51.100.7',
    );
  });

  it('reads the jumbo ceiling on a larger link as a lower bound, not a tunnel', async () => {
    await measure(result({ pathMtu: 9000, localMtu: 65536 }));

    expect(screen.getByTestId('path-mtu-value')).toHaveTextContent('At least 9000');
    expect(screen.getByTestId('path-mtu-verdict')).toHaveTextContent(
      'Every hop carries the largest size probed.',
    );
  });

  it('names a tunnel when the path carries less than the local link', async () => {
    await measure(result({ pathMtu: 1420 }));

    expect(screen.getByTestId('path-mtu-value')).toHaveTextContent('1420');
    expect(screen.getByTestId('path-mtu-verdict')).toHaveTextContent(/tunnel or VPN/);
  });

  it('gives a silent drop as a lower bound, not as the answer', async () => {
    await measure(result({ status: 'icmp_filtered', pathMtu: 1280 }));

    expect(screen.getByTestId('path-mtu-result')).toHaveAttribute('data-status', 'icmp_filtered');
    expect(screen.getByTestId('path-mtu-value')).toHaveTextContent('At least 1280');
    expect(screen.getByTestId('path-mtu-verdict')).toHaveTextContent(/drops them silently/);
  });

  it('shows no size for an unreachable destination', async () => {
    await measure(result({ status: 'unreachable', pathMtu: 0 }));

    expect(screen.queryByTestId('path-mtu-value')).not.toBeInTheDocument();
    expect(screen.getByTestId('path-mtu-verdict')).toHaveTextContent(/did not answer/);
  });

  it('keeps the bound reached when stopped early', async () => {
    jobs.cancelJob.mockResolvedValue(job('running'));
    jobs.submitJob.mockResolvedValue(job('running'));
    jobs.getJob.mockResolvedValue(job('running'));
    render(<PathMTUCard />);
    await userEvent.type(screen.getByTestId('path-mtu-target'), '198.51.100.7');
    await userEvent.click(screen.getByTestId('path-mtu-start'));
    await userEvent.click(await screen.findByTestId('path-mtu-stop'));
    act(() =>
      jobs.onJob(job('succeeded', { result: result({ status: 'incomplete', pathMtu: 1000 }) })),
    );

    expect(jobs.cancelJob).toHaveBeenCalledWith(JOB_ID);
    expect(screen.getByTestId('path-mtu-value')).toHaveTextContent('At least 1000');
    expect(screen.getByTestId('path-mtu-verdict')).toHaveTextContent(/Stopped before/);
  });
});
