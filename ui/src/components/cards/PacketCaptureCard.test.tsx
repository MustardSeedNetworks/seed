/**
 * PacketCaptureCard tests (#326, #239).
 *
 * The capture runs as a job, so these drive the card through the job client
 * and the job event stream: a viewer is never offered the form, a stop ends
 * in a summary rather than in nothing, and running, empty and failed read as
 * three different things.
 */

import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import type { JobResponse } from '../../types/generated/job-response';
import type { Result } from '../../types/generated/packet-capture-response';
import { PacketCaptureCard } from './PacketCaptureCard';

const role = vi.hoisted(() => ({ canWrite: true }));
vi.mock('../../contexts/RoleContext', () => ({
  useRole: () => ({ canWrite: role.canWrite }),
}));

const jobs = vi.hoisted(() => ({
  submitJob: vi.fn<(req: unknown) => Promise<JobResponse>>(),
  getJob: vi.fn<(id: string) => Promise<JobResponse>>(),
  cancelJob: vi.fn<(id: string) => Promise<JobResponse>>(),
  onJob: (_job: JobResponse): void => undefined,
  streamStatus: 'open',
}));
vi.mock('../../lib/jobsClient', () => ({
  submitJob: (req: unknown) => jobs.submitJob(req),
  getJob: (id: string) => jobs.getJob(id),
  cancelJob: (id: string) => jobs.cancelJob(id),
}));
vi.mock('../../hooks/useJobEvents', () => ({
  useJobEvents: (onJob: (job: JobResponse) => void) => {
    jobs.onJob = onJob;
    return { status: jobs.streamStatus };
  },
}));

const JOB_ID = 'job-1';

function job(state: string, extra: Partial<JobResponse> = {}): JobResponse {
  return { id: JOB_ID, kind: 'packet-capture', state, progress: 0, ...extra };
}

function captureResult(overrides: Partial<Result> = {}): Result {
  return {
    id: 'cap-abc123',
    interface: 'lo',
    packets: 42,
    bytes: 9000,
    durationMs: 3400,
    stopReason: 'stopped',
    summary: {
      protocols: [
        { name: 'TCP', packets: 30, bytes: 7000 },
        { name: 'DNS', packets: 12, bytes: 2000 },
      ],
      topTalkers: [{ address: '127.0.0.1', packets: 42, bytesSent: 4500, bytesReceived: 4500 }],
      topFlows: [
        {
          transport: 'TCP',
          source: '127.0.0.1:51000',
          destination: '127.0.0.1:8443',
          packets: 30,
          bytes: 7000,
          durationMs: 3000,
        },
      ],
      dnsQueries: 6,
      tcpConnections: 2,
      httpRequests: 0,
      truncated: false,
    },
    ...overrides,
  };
}

function emit(event: JobResponse): void {
  act(() => jobs.onJob(event));
}

async function startCapture(): Promise<void> {
  jobs.submitJob.mockResolvedValue(job('running'));
  jobs.getJob.mockResolvedValue(job('running'));
  await userEvent.click(screen.getByTestId('packet-capture-start'));
  await screen.findByTestId('packet-capture-stop');
}

beforeEach(async () => {
  role.canWrite = true;
  jobs.streamStatus = 'open';
  jobs.submitJob.mockReset();
  jobs.getJob.mockReset();
  jobs.cancelJob.mockReset();
  await i18n.changeLanguage('en');
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('PacketCaptureCard', () => {
  it('gives a viewer the reason instead of a start control', () => {
    role.canWrite = false;
    render(<PacketCaptureCard defaultInterface="lo" />);

    expect(screen.queryByTestId('packet-capture-start')).not.toBeInTheDocument();
    expect(screen.queryByTestId('packet-capture-interface')).not.toBeInTheDocument();
    expect(screen.getByTestId('packet-capture-read-only')).toHaveTextContent(/operator role/);
    expect(jobs.submitJob).not.toHaveBeenCalled();
  });

  it('starts on the current interface with the duration asked for', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);

    const duration = screen.getByTestId('packet-capture-duration');
    await userEvent.clear(duration);
    await userEvent.type(duration, '5');
    await startCapture();

    // A blank filter is not sent: an empty one would be a filter that keeps nothing.
    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'packet-capture',
      params: { interface: 'lo', filter: undefined, durationSeconds: 5 },
    });
    expect(screen.getByTestId('packet-capture')).toHaveAttribute('data-phase', 'running');
    expect(screen.getByTestId('packet-capture-running')).toHaveTextContent(
      'Capturing on lo. Stops after 5 s or at 64 MB',
    );
  });

  it('sends the filter and an interface the operator typed', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);

    const iface = screen.getByTestId('packet-capture-interface');
    await userEvent.clear(iface);
    await userEvent.type(iface, 'eth1');
    await userEvent.type(screen.getByTestId('packet-capture-filter'), 'port 53');
    await startCapture();

    expect(jobs.submitJob).toHaveBeenCalledWith({
      kind: 'packet-capture',
      params: { interface: 'eth1', filter: 'port 53', durationSeconds: 60 },
    });
  });

  it('refuses a duration outside the job bounds', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);

    const duration = screen.getByTestId('packet-capture-duration');
    await userEvent.clear(duration);
    await userEvent.type(duration, '3601');

    expect(screen.getByTestId('packet-capture-start')).toBeDisabled();
    expect(screen.getByText('Enter a whole number from 1 to 3600.')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('packet-capture-start'));
    expect(jobs.submitJob).not.toHaveBeenCalled();
  });

  it('stops through the job and shows what the stopped capture recorded', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    jobs.cancelJob.mockResolvedValue(job('running'));
    await userEvent.click(screen.getByTestId('packet-capture-stop'));
    expect(jobs.cancelJob).toHaveBeenCalledWith(JOB_ID);
    expect(screen.getByTestId('packet-capture')).toHaveAttribute('data-phase', 'stopping');
    expect(screen.getByTestId('packet-capture-stop')).toBeDisabled();

    // A stopped capture keeps its file: the job ends succeeded, not cancelled.
    emit(job('succeeded', { result: captureResult() }));

    const result = screen.getByTestId('packet-capture-result');
    expect(result).toHaveAttribute('data-stop-reason', 'stopped');
    expect(screen.getByTestId('packet-capture-totals')).toHaveTextContent(
      '42 packets, 8.8 KB in 3 s on lo',
    );
    expect(result).toHaveTextContent('Stopped early.');
    expect(screen.getByTestId('packet-capture-download')).toHaveAttribute(
      'href',
      '/api/v1/captures/cap-abc123',
    );
    const protocols = within(screen.getByTestId('packet-capture-protocols'));
    expect(protocols.getByText('TCP')).toBeInTheDocument();
    expect(protocols.getByText('DNS')).toBeInTheDocument();
    expect(
      within(screen.getByTestId('packet-capture-talkers')).getByText('127.0.0.1'),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId('packet-capture-flows')).getByText('127.0.0.1:8443'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('packet-capture-counters')).toHaveTextContent('DNS queries6');
    // The form is back for the next capture.
    expect(screen.getByTestId('packet-capture-start')).toBeEnabled();
  });

  it('reads back a capture that ended before its start request returned', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);

    let resolveSubmit: (value: JobResponse) => void = () => undefined;
    jobs.submitJob.mockReturnValue(
      new Promise((resolve) => {
        resolveSubmit = resolve;
      }),
    );
    jobs.getJob.mockResolvedValue(
      job('succeeded', { result: captureResult({ stopReason: 'duration' }) }),
    );
    await userEvent.click(screen.getByTestId('packet-capture-start'));
    // Its end went out on the stream while the ID was not yet known.
    emit(job('succeeded', { result: captureResult({ stopReason: 'duration' }) }));
    await act(async () => resolveSubmit(job('queued')));

    expect(jobs.getJob).toHaveBeenCalledWith(JOB_ID);
    expect(screen.getByTestId('packet-capture-result')).toHaveAttribute(
      'data-stop-reason',
      'duration',
    );
  });

  it('reads the job back when the stream connects after the capture ended', async () => {
    jobs.streamStatus = 'connecting';
    const { rerender } = render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    // Published before the stream was listening, so never delivered.
    jobs.getJob.mockResolvedValue(job('failed', { error: 'capture: interface went away' }));
    jobs.streamStatus = 'open';
    rerender(<PacketCaptureCard defaultInterface="lo" />);

    await waitFor(() => {
      expect(screen.getByTestId('packet-capture')).toHaveAttribute('data-phase', 'failed');
    });
    expect(screen.getByTestId('packet-capture-error-detail')).toHaveTextContent(
      'capture: interface went away',
    );
  });

  it('ignores other jobs on the shared stream', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    emit({ ...job('failed', { error: 'not ours' }), id: 'job-2', kind: 'engine-scan' });

    expect(screen.getByTestId('packet-capture')).toHaveAttribute('data-phase', 'running');
  });

  it('takes the end a stop request answers with when the capture had already ended', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    jobs.cancelJob.mockResolvedValue(
      job('succeeded', { result: captureResult({ stopReason: 'duration' }) }),
    );
    await userEvent.click(screen.getByTestId('packet-capture-stop'));

    expect(screen.getByTestId('packet-capture-result')).toHaveAttribute(
      'data-stop-reason',
      'duration',
    );
  });

  it('says a capture that matched nothing is empty, not failed', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    emit(
      job('succeeded', {
        result: captureResult({
          packets: 0,
          bytes: 0,
          stopReason: 'duration',
          summary: {
            protocols: [],
            topTalkers: [],
            topFlows: [],
            dnsQueries: 0,
            tcpConnections: 0,
            httpRequests: 0,
            truncated: false,
          },
        }),
      }),
    );

    expect(screen.getByTestId('packet-capture-empty')).toBeInTheDocument();
    expect(screen.queryByTestId('packet-capture-protocols')).not.toBeInTheDocument();
    expect(screen.queryByTestId('packet-capture-failed')).not.toBeInTheDocument();
    expect(screen.getByTestId('packet-capture-result')).toHaveTextContent(
      'Ran for the full duration.',
    );
  });

  it('shows a failed capture with the reason the daemon gave', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    emit(job('failed', { error: 'open eth9: no such device' }));

    expect(screen.getByTestId('packet-capture')).toHaveAttribute('data-phase', 'failed');
    expect(screen.getByRole('alert')).toHaveTextContent('The capture did not run.');
    expect(screen.getByTestId('packet-capture-error-detail')).toHaveTextContent(
      'open eth9: no such device',
    );
    expect(screen.queryByTestId('packet-capture-result')).not.toBeInTheDocument();
  });

  it('says so when the start request itself is refused', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    jobs.submitJob.mockRejectedValue(new Error('Forbidden'));

    await userEvent.click(screen.getByTestId('packet-capture-start'));

    await waitFor(() => {
      expect(screen.getByTestId('packet-capture')).toHaveAttribute('data-phase', 'failed');
    });
    expect(screen.getByTestId('packet-capture-error-detail')).toHaveTextContent('Forbidden');
  });

  it('keeps running and says so when the stop is refused', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    jobs.cancelJob.mockRejectedValue(new Error('Service Unavailable'));
    await userEvent.click(screen.getByTestId('packet-capture-stop'));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('The capture did not stop. Try again.');
    });
    expect(screen.getByTestId('packet-capture-stop')).toBeEnabled();
  });

  it('notes a summary that ran out of room', async () => {
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    const result = captureResult();
    emit(
      job('succeeded', { result: { ...result, summary: { ...result.summary, truncated: true } } }),
    );

    expect(screen.getByTestId('packet-capture-truncated')).toBeInTheDocument();
  });

  it('renders Spanish under es', async () => {
    await i18n.changeLanguage('es');
    render(<PacketCaptureCard defaultInterface="lo" />);
    await startCapture();

    emit(job('succeeded', { result: captureResult({ packets: 1 }) }));

    expect(screen.getByTestId('packet-capture-totals')).toHaveTextContent(
      '1 paquete, 8.8 KB en 3 s en lo',
    );
    expect(screen.getByTestId('packet-capture-result')).toHaveTextContent(
      'Se detuvo antes de tiempo.',
    );
    expect(screen.getByTestId('packet-capture-download')).toHaveTextContent('Descargar PCAP');
  });
});
