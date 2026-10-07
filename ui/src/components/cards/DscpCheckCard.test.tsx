/**
 * DscpCheckCard tests (#400).
 *
 * The check runs as a job, so these drive the card through the job client and
 * the job event stream: below Pro the card says so and sends nothing, a viewer
 * is never offered the form, each mode submits its own job kind, and the
 * per-class verdicts read as pass or fail.
 */

import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import type { JobResponse } from '../../types/generated/job-response';
import type { ClassResult, ListenResult } from '../../types/generated/qos-listen-response';
import type { SingleHostResult } from '../../types/generated/qos-single-host-response';
import { DscpCheckCard } from './DscpCheckCard';

const role = vi.hoisted(() => ({ canWrite: true }));
vi.mock('../../contexts/RoleContext', () => ({
  useRole: () => ({ canWrite: role.canWrite }),
}));

const licence = vi.hoisted(() => ({ features: ['dscp_verification'], loading: false }));
vi.mock('../../contexts/LicenseContext', () => ({
  useLicense: () => ({
    loading: licence.loading,
    hasFeature: (feature: string) => licence.features.includes(feature),
  }),
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
  return { id: JOB_ID, kind: 'qos-listen', state, progress: 0, ...extra };
}

function kept(dscp: number, name: string): ClassResult {
  return {
    sentDscp: dscp,
    sentName: name,
    expected: 5,
    received: 5,
    observed: [{ dscp, name, count: 5 }],
    verdict: 'preserved',
  };
}

/** EF rewritten to best effort and AF41 dropped, the faults the card exists for. */
const FAULTY: ClassResult[] = [
  {
    sentDscp: 46,
    sentName: 'EF',
    expected: 5,
    received: 5,
    observed: [{ dscp: 0, name: 'CS0', count: 5 }],
    verdict: 'remarked',
  },
  { sentDscp: 34, sentName: 'AF41', expected: 5, received: 0, observed: [], verdict: 'lost' },
  kept(0, 'CS0'),
];

function listenResult(overrides: Partial<ListenResult> = {}): ListenResult {
  return {
    port: 45400,
    family: 'ipv4',
    listenedMs: 10000,
    probes: 15,
    ignored: 0,
    runs: [
      {
        runId: 'a1b2c3d4e5f60718',
        sender: '192.0.2.20',
        classes: [kept(46, 'EF'), kept(0, 'CS0')],
        preserved: true,
      },
    ],
    runsTruncated: false,
    dscpObserved: true,
    ...overrides,
  };
}

function emit(event: JobResponse): void {
  act(() => jobs.onJob(event));
}

async function startRunning(): Promise<void> {
  jobs.submitJob.mockResolvedValue(job('running'));
  jobs.getJob.mockResolvedValue(job('running'));
  await userEvent.click(screen.getByTestId('dscp-check-start'));
  await screen.findByTestId('dscp-check-stop');
}

beforeEach(async () => {
  role.canWrite = true;
  licence.features = ['dscp_verification'];
  licence.loading = false;
  jobs.submitJob.mockReset();
  jobs.getJob.mockReset();
  jobs.cancelJob.mockReset();
  await i18n.changeLanguage('en');
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('DscpCheckCard', () => {
  it('below Pro shows the upgrade prompt and no form', () => {
    licence.features = ['path_analysis'];
    render(<DscpCheckCard defaultInterface="eth0" />);

    expect(screen.getByTestId('dscp-check-upgrade')).toHaveTextContent(
      'This feature requires the Pro tier',
    );
    expect(screen.queryByTestId('dscp-check-start')).not.toBeInTheDocument();
    expect(screen.queryByTestId('dscp-check-port')).not.toBeInTheDocument();
    expect(jobs.submitJob).not.toHaveBeenCalled();
  });

  it('shows neither the prompt nor the form while the licence loads', () => {
    licence.loading = true;
    licence.features = [];
    render(<DscpCheckCard defaultInterface="eth0" />);

    expect(screen.queryByTestId('dscp-check-upgrade')).not.toBeInTheDocument();
    expect(screen.queryByTestId('dscp-check-start')).not.toBeInTheDocument();
  });

  it('gives a viewer the reason instead of a start control', () => {
    role.canWrite = false;
    render(<DscpCheckCard defaultInterface="eth0" />);

    expect(screen.getByTestId('dscp-check-read-only')).toHaveTextContent(/operator role/);
    expect(screen.queryByTestId('dscp-check-start')).not.toBeInTheDocument();
  });

  it('listens on the shared default port for ten seconds with nothing typed', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    await startRunning();

    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'qos-listen',
      params: { port: 45400, durationSeconds: 10 },
    });
    expect(screen.getByTestId('dscp-check-running')).toHaveTextContent(
      'Listening. Start Send on the other Seed now.',
    );
  });

  it('refuses a port or duration outside the job bounds', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    const start = screen.getByTestId('dscp-check-start');

    const duration = screen.getByTestId('dscp-check-duration');
    await userEvent.clear(duration);
    await userEvent.type(duration, '61');
    expect(start).toBeDisabled();
    expect(screen.getByText('Enter a whole number from 1 to 60.')).toBeInTheDocument();

    await userEvent.clear(duration);
    await userEvent.type(duration, '30');
    const port = screen.getByTestId('dscp-check-port');
    await userEvent.clear(port);
    await userEvent.type(port, '70000');
    expect(start).toBeDisabled();
    expect(screen.getByText('Enter a whole number from 1 to 65535.')).toBeInTheDocument();
  });

  it('sends to the listening Seed and points at it for the verdict', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    await userEvent.click(screen.getByTestId('dscp-check-mode-send'));
    const start = screen.getByTestId('dscp-check-start');
    expect(start).toBeDisabled();

    await userEvent.type(screen.getByTestId('dscp-check-target'), '192.0.2.10');
    await startRunning();
    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'qos-send',
      params: { target: '192.0.2.10', port: 45400 },
    });
    // The mode cannot change under a running check.
    expect(within(screen.getByTestId('dscp-check-mode-listen')).getByRole('radio')).toBeDisabled();

    emit(
      job('succeeded', {
        result: {
          target: '192.0.2.10',
          port: 45400,
          runId: 'a1b2c3d4e5f60718',
          count: 5,
          classes: [
            { dscp: 46, name: 'EF', sent: 5 },
            { dscp: 0, name: 'CS0', sent: 5 },
          ],
          marked: false,
        },
      }),
    );

    expect(screen.getByTestId('dscp-check-sent')).toHaveTextContent(
      'Sent 5 probes of each class (EF (46), CS0 (0)) to 192.0.2.10 port 45400. Read the verdict on the listening Seed.',
    );
    expect(screen.getByTestId('dscp-check-unmarked')).toBeInTheDocument();
  });

  it('reads a listen where every class kept its marking as a pass', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    await startRunning();
    emit(job('succeeded', { result: listenResult() }));

    expect(screen.getByTestId('dscp-check-run')).toHaveTextContent('From 192.0.2.20');
    expect(screen.getByTestId('dscp-check-verdict')).toHaveAttribute('data-pass', 'true');
    expect(screen.getByTestId('dscp-check-verdict')).toHaveTextContent(/^Pass:/);
  });

  it('names each changed or lost class in a failing listen', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    await startRunning();
    emit(
      job('succeeded', {
        result: listenResult({
          runs: [{ runId: 'r', sender: '192.0.2.20', classes: FAULTY, preserved: false }],
        }),
      }),
    );

    expect(screen.getByTestId('dscp-check-verdict')).toHaveTextContent(
      'Fail: 2 of 3 classes were changed or lost on the way.',
    );
    const rows = within(screen.getByTestId('dscp-check-classes')).getAllByRole('row');
    expect(rows[1]).toHaveTextContent('EF (46)ChangedCS0 (0) ×55/5');
    expect(rows[2]).toHaveTextContent('AF41 (34)Lost—0/5');
    expect(rows[3]).toHaveTextContent('CS0 (0)KeptCS0 (0) ×55/5');
  });

  it('says no probes arrived rather than that the check failed', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    await startRunning();
    emit(job('succeeded', { result: listenResult({ runs: [], probes: 0 }) }));

    expect(screen.getByTestId('dscp-check-silent')).toHaveTextContent(
      'No probes reached port 45400.',
    );
    expect(screen.queryByTestId('dscp-check-failed')).not.toBeInTheDocument();
  });

  it('does not call a listen that cannot read markings a failure', async () => {
    const unread: ClassResult = { ...kept(46, 'EF'), observed: [], verdict: 'unobserved' };
    render(<DscpCheckCard defaultInterface="eth0" />);
    await startRunning();
    emit(
      job('succeeded', {
        result: listenResult({
          dscpObserved: false,
          runs: [{ runId: 'r', sender: '192.0.2.20', classes: [unread], preserved: false }],
        }),
      }),
    );

    expect(screen.getByTestId('dscp-check-verdict')).toHaveTextContent(/cannot read the marking/);
    expect(screen.getByTestId('dscp-check-verdict')).not.toHaveAttribute('data-pass');
  });

  it('runs the one-host check across two different interfaces', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    await userEvent.click(screen.getByTestId('dscp-check-mode-single-host'));
    const start = screen.getByTestId('dscp-check-start');

    await userEvent.type(screen.getByTestId('dscp-check-send-interface'), 'eth0');
    expect(start).toBeDisabled();
    expect(screen.getByText('Choose two different interfaces.')).toBeInTheDocument();

    const send = screen.getByTestId('dscp-check-send-interface');
    await userEvent.clear(send);
    await userEvent.type(send, 'wlan0');
    await startRunning();
    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'qos-single-host',
      params: { sendInterface: 'wlan0', captureInterface: 'eth0' },
    });

    const result: SingleHostResult = {
      sendInterface: 'wlan0',
      captureInterface: 'eth0',
      source: '192.0.2.31',
      target: '192.0.2.30',
      nextHop: '192.0.2.1',
      routed: true,
      port: 41234,
      runId: 'r',
      count: 5,
      probes: 10,
      classes: FAULTY,
      preserved: false,
    };
    emit(job('succeeded', { result }));

    expect(screen.getByTestId('dscp-check-route')).toHaveTextContent(
      'Out of wlan0 (192.0.2.31) through 192.0.2.1, back in on eth0 (192.0.2.30).',
    );
    expect(screen.getByTestId('dscp-check-verdict')).toHaveAttribute('data-pass', 'false');
  });

  it('leaves a result behind when the mode changes', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    await startRunning();
    emit(job('succeeded', { result: listenResult() }));
    expect(screen.getByTestId('dscp-check-result')).toBeInTheDocument();

    await userEvent.click(screen.getByTestId('dscp-check-mode-send'));
    expect(screen.queryByTestId('dscp-check-result')).not.toBeInTheDocument();
  });

  it('shows a refused start with the reason the daemon gave', async () => {
    render(<DscpCheckCard defaultInterface="eth0" />);
    jobs.submitJob.mockRejectedValue(new Error('Payment Required'));
    await userEvent.click(screen.getByTestId('dscp-check-start'));

    await waitFor(() => {
      expect(screen.getByTestId('dscp-check')).toHaveAttribute('data-phase', 'failed');
    });
    expect(screen.getByTestId('dscp-check-error-detail')).toHaveTextContent('Payment Required');
  });

  it('renders Spanish under es', async () => {
    await i18n.changeLanguage('es');
    render(<DscpCheckCard defaultInterface="eth0" />);
    await startRunning();
    emit(job('succeeded', { result: listenResult() }));

    expect(screen.getByTestId('dscp-check-verdict')).toHaveTextContent(/^Correcto:/);
    expect(screen.getByTestId('dscp-check-run')).toHaveTextContent('Desde 192.0.2.20');
  });
});
