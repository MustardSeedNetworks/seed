/**
 * useBoundedJob — run one bounded job through the jobs spine and keep what it
 * found when the operator stops it.
 *
 * POST /jobs starts the job, the job event stream reports its end (GET
 * /jobs/{id} fills what the stream missed), and DELETE /jobs/{id} stops it
 * early. The kinds this serves keep what they recorded when stopped, so a stop
 * ends in `succeeded` with a partial result, not in `cancelled`.
 */

import { useEffect, useRef, useState } from 'react';
import { cancelJob, getJob, submitJob } from '../lib/jobsClient';
import type { JobResponse } from '../types/generated/job-response';
import { useJobEvents } from './useJobEvents';

/**
 * The card's view of one run. `starting` covers the POST, `stopping` the wait
 * between the stop request and the job's end.
 */
export type BoundedJobState<Result> =
  | { phase: 'idle' }
  | { phase: 'starting' }
  | { phase: 'running'; jobId: string; stopFailed?: boolean }
  | { phase: 'stopping'; jobId: string }
  | { phase: 'finished'; result: Result }
  | { phase: 'failed'; error: string };

export interface UseBoundedJobReturn<Request, Result> {
  state: BoundedJobState<Result>;
  start: (request: Request) => Promise<void>;
  stop: () => Promise<void>;
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function useBoundedJob<Request, Result>(
  kind: string,
  isResult: (result: unknown) => result is Result,
): UseBoundedJobReturn<Request, Result> {
  const [state, setState] = useState<BoundedJobState<Result>>({ phase: 'idle' });
  const jobIdRef = useRef('');

  const apply = (job: JobResponse): void => {
    if (job.id !== jobIdRef.current) {
      return;
    }
    switch (job.state) {
      case 'succeeded':
        jobIdRef.current = '';
        setState(
          isResult(job.result)
            ? { phase: 'finished', result: job.result }
            : { phase: 'failed', error: '' },
        );
        return;
      case 'failed':
      case 'cancelled':
        jobIdRef.current = '';
        setState({ phase: 'failed', error: job.error ?? '' });
        return;
      default:
        setState((prev) =>
          prev.phase === 'stopping' ? prev : { phase: 'running', jobId: job.id },
        );
    }
  };

  // The stream is live-only: a change published before it connects, or while
  // it reconnects, is never delivered. The runner's own answer is to re-read
  // the job, so the hook does that whenever it may have missed one.
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

  const start = async (request: Request): Promise<void> => {
    setState({ phase: 'starting' });
    try {
      const job = await submitJob({ kind, params: request });
      jobIdRef.current = job.id;
      apply(job);
    } catch (err) {
      jobIdRef.current = '';
      setState({ phase: 'failed', error: errorMessage(err) });
      return;
    }
    // A job that ends within the POST's round trip has already published
    // its end.
    await resync();
  };

  const stop = async (): Promise<void> => {
    const jobId = jobIdRef.current;
    if (!jobId) {
      return;
    }
    setState({ phase: 'stopping', jobId });
    try {
      // A job that ended as the stop was sent answers with its end.
      apply(await cancelJob(jobId));
    } catch {
      // The job is still running, so the operator can try again.
      setState({ phase: 'running', jobId, stopFailed: true });
    }
  };

  return { state, start, stop };
}
