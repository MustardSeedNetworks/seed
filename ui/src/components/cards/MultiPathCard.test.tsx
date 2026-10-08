/**
 * MultiPathCard tests (#395).
 *
 * The check runs as a `path-multipath` job, so these drive the card through
 * the job client and the job event stream: a viewer is never offered the form,
 * a run needs a destination, the routes render with the hop they split at,
 * a stop keeps what was traced, and a refused run shows the server's reason.
 */

import { act, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { MultiPathResult, PathVariant } from '../../hooks/usePathChecks';
import i18n from '../../i18n';
import type { JobResponse } from '../../types/generated/job-response';
import { MultiPathCard } from './MultiPathCard';

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
  return { id: JOB_ID, kind: 'path-multipath', state, progress: 0, ...extra };
}

const REACHED: PathVariant = {
  hops: ['10.0.0.1', '10.1.0.1', '198.51.100.7'],
  seen: 5,
  completed: true,
};
const SILENT: PathVariant = { hops: ['10.0.0.1', '10.2.0.1', ''], seen: 3, completed: false };

const SPLIT: MultiPathResult = {
  target: '198.51.100.7',
  targetIp: '198.51.100.7',
  attempts: 8,
  paths: [REACHED, SILENT],
  divergesAtTtl: 2,
};

function route(index: number): HTMLElement {
  const found = screen.getAllByTestId('multi-path-route')[index];
  if (!found) {
    throw new Error(`no route ${index + 1}`);
  }
  return found;
}

async function run(destination: string): Promise<void> {
  jobs.submitJob.mockResolvedValue(job('running'));
  jobs.getJob.mockResolvedValue(job('running'));
  await userEvent.type(screen.getByTestId('multi-path-target'), destination);
  await userEvent.click(screen.getByTestId('multi-path-start'));
  await screen.findByTestId('multi-path-stop');
}

function finish(result: unknown): void {
  act(() => jobs.onJob(job('succeeded', { result })));
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

describe('MultiPathCard', () => {
  it('gives a viewer the reason instead of a start control', () => {
    role.canWrite = false;
    render(<MultiPathCard />);

    expect(screen.queryByTestId('multi-path-start')).not.toBeInTheDocument();
    expect(screen.getByTestId('multi-path-read-only')).toHaveTextContent(/operator role/);
  });

  it('waits for a destination, then submits only the destination', async () => {
    render(<MultiPathCard />);
    expect(screen.getByTestId('multi-path-start')).toBeDisabled();

    await run(' 198.51.100.7 ');

    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'path-multipath',
      params: { destination: '198.51.100.7' },
    });
    expect(screen.getByTestId('multi-path')).toHaveAttribute('data-phase', 'running');
    expect(screen.getByTestId('multi-path-target')).toBeDisabled();
  });

  it('lists each route, how often it was taken, and marks the hop where they split', async () => {
    render(<MultiPathCard />);
    await run('198.51.100.7');
    finish(SPLIT);

    expect(screen.getByTestId('multi-path-summary')).toHaveTextContent(
      '2 routes to 198.51.100.7. They split at hop 2.',
    );
    expect(screen.getAllByTestId('multi-path-route')).toHaveLength(2);
    const first = route(0);
    const second = route(1);
    expect(within(first).getByTestId('multi-path-route-seen')).toHaveTextContent(
      'Route 1: 5 of 8 traces',
    );
    expect(first).toHaveAttribute('data-completed', 'true');
    expect(within(second).getByTestId('multi-path-route-seen')).toHaveTextContent(
      'Route 2: 3 of 8 traces · did not reach the destination',
    );
    const hops = within(second).getAllByTestId('multi-path-hop');
    expect(hops.map((hop) => hop.textContent)).toEqual(['110.0.0.1', '210.2.0.1', '3*']);
    expect(hops[1]).toHaveAttribute('data-split', 'true');
    expect(hops[0]).not.toHaveAttribute('data-split');
    expect(screen.getByTestId('multi-path-start')).toBeEnabled();
  });

  it('reads a single route as one route, not as a split', async () => {
    render(<MultiPathCard />);
    await run('198.51.100.7');
    finish({ ...SPLIT, paths: [{ ...REACHED, seen: 8 }], divergesAtTtl: 0 });

    expect(screen.getByTestId('multi-path-summary')).toHaveTextContent(
      'One route to 198.51.100.7 in 8 traces.',
    );
    expect(screen.queryByText(/split/)).not.toBeInTheDocument();
  });

  it('keeps what was traced when stopped', async () => {
    jobs.cancelJob.mockResolvedValue(job('running'));
    render(<MultiPathCard />);
    await run('198.51.100.7');

    await userEvent.click(screen.getByTestId('multi-path-stop'));
    expect(jobs.cancelJob).toHaveBeenCalledWith(JOB_ID);
    finish({ ...SPLIT, attempts: 3, paths: [{ ...REACHED, seen: 3 }], divergesAtTtl: 0 });

    expect(screen.getByTestId('multi-path')).toHaveAttribute('data-phase', 'finished');
    expect(screen.getByTestId('multi-path-summary')).toHaveTextContent('in 3 traces');
  });

  it('shows the server reason when the run is refused', async () => {
    render(<MultiPathCard />);
    await run('nowhere.invalid');
    act(() => jobs.onJob(job('failed', { error: 'lookup nowhere.invalid: no such host' })));

    expect(screen.getByTestId('multi-path-failed')).toHaveTextContent(
      'The multi-path check did not run.',
    );
    expect(screen.getByTestId('multi-path-error-detail')).toHaveTextContent('no such host');
    expect(screen.queryByTestId('multi-path-result')).not.toBeInTheDocument();
  });
});
