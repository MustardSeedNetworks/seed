/**
 * usePathMonitor — run one continuous path monitor through the jobs spine
 * (#165), the mtr view of a path.
 *
 * A monitor is a `path-monitor` job with no natural end: POST /jobs starts it
 * and DELETE /jobs/{id} stops it. Each round's accumulated per-hop view is
 * broadcast as a `pathMonitor` frame on the main event stream, correlated by
 * destination (the job is not told its own id). A stopped monitor keeps what
 * it measured, so stopping ends in `succeeded` with the final snapshot as the
 * job's result — not in `cancelled`.
 */

import { useEffect, useRef, useState } from 'react';
import { cancelJob, getJob, submitJob } from '../lib/jobsClient';
import type { JobResponse } from '../types/generated/job-response';
import type { PathMonitorRequest } from '../types/generated/path-monitor-request';
import type { PathMonitorUpdate } from '../types/generated/path-monitor-update';
import { useJobEvents } from './useJobEvents';

export type { PathMonitorRequest, PathMonitorUpdate };

/**
 * The card's view of one monitor. `starting` covers the POST, `stopping` the
 * wait between the stop request and the job's end. `snapshot` is the latest
 * round, absent until the first one lands.
 */
export type PathMonitorState =
  | { phase: 'idle' }
  | { phase: 'starting'; target: string }
  | {
      phase: 'running';
      jobId: string;
      target: string;
      snapshot?: PathMonitorUpdate;
      stopFailed?: boolean;
    }
  | { phase: 'stopping'; jobId: string; target: string; snapshot?: PathMonitorUpdate }
  | { phase: 'stopped'; snapshot: PathMonitorUpdate }
  | { phase: 'failed'; error: string; snapshot?: PathMonitorUpdate };

export interface UsePathMonitorReturn {
  state: PathMonitorState;
  start: (request: PathMonitorRequest) => Promise<void>;
  stop: () => Promise<void>;
}

const PATH_MONITOR_KIND = 'path-monitor';

/** isPathMonitorUpdate narrows an SSE payload or job result to a snapshot. */
export function isPathMonitorUpdate(value: unknown): value is PathMonitorUpdate {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as { target?: unknown }).target === 'string' &&
    typeof (value as { rounds?: unknown }).rounds === 'number' &&
    Array.isArray((value as { hops?: unknown }).hops)
  );
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function latestSnapshot(state: PathMonitorState): PathMonitorUpdate | undefined {
  return state.phase === 'idle' || state.phase === 'starting' ? undefined : state.snapshot;
}

export function usePathMonitor(
  subscribe: (handler: (update: PathMonitorUpdate) => void) => () => void,
): UsePathMonitorReturn {
  const [state, setState] = useState<PathMonitorState>({ phase: 'idle' });
  const jobIdRef = useRef('');
  const targetRef = useRef('');

  useEffect(
    () =>
      subscribe((update) => {
        if (!jobIdRef.current || update.target !== targetRef.current) {
          return;
        }
        setState((prev) =>
          prev.phase === 'running' || prev.phase === 'stopping'
            ? { ...prev, snapshot: update }
            : prev,
        );
      }),
    [subscribe],
  );

  const apply = (job: JobResponse): void => {
    if (job.id !== jobIdRef.current) {
      return;
    }
    switch (job.state) {
      case 'succeeded':
        jobIdRef.current = '';
        setState((prev) =>
          isPathMonitorUpdate(job.result)
            ? { phase: 'stopped', snapshot: job.result }
            : { phase: 'failed', error: '', snapshot: latestSnapshot(prev) },
        );
        return;
      case 'failed':
      case 'cancelled':
        jobIdRef.current = '';
        setState((prev) => ({
          phase: 'failed',
          error: job.error ?? '',
          snapshot: latestSnapshot(prev),
        }));
        return;
      default:
        setState((prev) =>
          prev.phase === 'running' || prev.phase === 'stopping'
            ? prev
            : { phase: 'running', jobId: job.id, target: targetRef.current },
        );
    }
  };

  // The job stream is live-only: an end published before it connects, or
  // while it reconnects, is never delivered, so re-read the job then.
  const resync = async (): Promise<void> => {
    const jobId = jobIdRef.current;
    if (!jobId) {
      return;
    }
    try {
      apply(await getJob(jobId));
    } catch {
      // The next stream event or re-sync will bring the job's state.
    }
  };

  const { status: streamStatus } = useJobEvents(apply);

  useEffect(() => {
    if (streamStatus === 'open') {
      void resync();
    }
  }, [streamStatus, resync]);

  const start = async (request: PathMonitorRequest): Promise<void> => {
    targetRef.current = request.destination;
    setState({ phase: 'starting', target: request.destination });
    try {
      const job = await submitJob({ kind: PATH_MONITOR_KIND, params: request });
      jobIdRef.current = job.id;
      apply(job);
    } catch (err) {
      jobIdRef.current = '';
      setState({ phase: 'failed', error: errorMessage(err) });
      return;
    }
    // A monitor refused by its handler has already ended by now.
    await resync();
  };

  const stop = async (): Promise<void> => {
    const jobId = jobIdRef.current;
    if (!jobId) {
      return;
    }
    setState((prev) => ({
      phase: 'stopping',
      jobId,
      target: targetRef.current,
      snapshot: latestSnapshot(prev),
    }));
    try {
      apply(await cancelJob(jobId));
    } catch {
      // The monitor is still running, so the operator can try again.
      setState((prev) => ({
        phase: 'running',
        jobId,
        target: targetRef.current,
        snapshot: latestSnapshot(prev),
        stopFailed: true,
      }));
    }
  };

  // Leaving the page stops the monitor; the server's idle guard only covers a
  // tab that closed without saying so.
  useEffect(
    () => () => {
      const jobId = jobIdRef.current;
      if (jobId) {
        jobIdRef.current = '';
        void cancelJob(jobId).catch(() => undefined);
      }
    },
    [],
  );

  return { state, start, stop };
}
