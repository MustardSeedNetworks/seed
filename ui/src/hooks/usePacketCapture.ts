/**
 * usePacketCapture — run one packet capture through the jobs spine (#326).
 *
 * A capture is a `packet-capture` job: POST /jobs starts it, the job event
 * stream reports its end (GET /jobs/{id} fills what the stream missed), and
 * DELETE /jobs/{id} stops it early. A stopped
 * capture keeps what it recorded, so stopping ends in `succeeded` with a
 * result whose stopReason is "stopped" — not in `cancelled`. The result names
 * the file GET /api/v1/captures/{id} downloads, and carries its summary (#239).
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { cancelJob, getJob, submitJob } from '../lib/jobsClient';
import type { JobResponse } from '../types/generated/job-response';
import type { Request as CaptureRequest } from '../types/generated/packet-capture-request';
import type { Result } from '../types/generated/packet-capture-response';
import { useJobEvents } from './useJobEvents';

export type { CaptureRequest };
export type CaptureResult = Result;

/**
 * The card's view of one capture. `starting` covers the POST, `stopping` the
 * wait between the stop request and the job's end.
 */
export type CaptureState =
  | { phase: 'idle' }
  | { phase: 'starting' }
  | { phase: 'running'; jobId: string; stopFailed?: boolean }
  | { phase: 'stopping'; jobId: string }
  | { phase: 'finished'; result: CaptureResult }
  | { phase: 'failed'; error: string };

export interface UsePacketCaptureReturn {
  state: CaptureState;
  start: (request: CaptureRequest) => Promise<void>;
  stop: () => Promise<void>;
}

const PACKET_CAPTURE_KIND = 'packet-capture';

function isCaptureResult(result: unknown): result is CaptureResult {
  return (
    typeof result === 'object' &&
    result !== null &&
    typeof (result as { id?: unknown }).id === 'string' &&
    typeof (result as { summary?: unknown }).summary === 'object'
  );
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function usePacketCapture(): UsePacketCaptureReturn {
  const [state, setState] = useState<CaptureState>({ phase: 'idle' });
  const jobIdRef = useRef('');

  const apply = useCallback((job: JobResponse): void => {
    if (job.id !== jobIdRef.current) {
      return;
    }
    switch (job.state) {
      case 'succeeded':
        jobIdRef.current = '';
        setState(
          isCaptureResult(job.result)
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
  }, []);

  // The stream is live-only: a change published before it connects, or while
  // it reconnects, is never delivered. The runner's own answer is to re-read
  // the job, so the hook does that whenever it may have missed one.
  const resync = useCallback(async (): Promise<void> => {
    const jobId = jobIdRef.current;
    if (!jobId) {
      return;
    }
    try {
      apply(await getJob(jobId));
    } catch {
      // The next stream event or re-sync will bring the job's state.
    }
  }, [apply]);

  const { status: streamStatus } = useJobEvents(apply);

  useEffect(() => {
    if (streamStatus === 'open') {
      void resync();
    }
  }, [streamStatus, resync]);

  const start = useCallback(
    async (request: CaptureRequest): Promise<void> => {
      setState({ phase: 'starting' });
      try {
        const job = await submitJob({ kind: PACKET_CAPTURE_KIND, params: request });
        jobIdRef.current = job.id;
        apply(job);
      } catch (err) {
        jobIdRef.current = '';
        setState({ phase: 'failed', error: errorMessage(err) });
        return;
      }
      // A capture that ends within the POST's round trip has already
      // published its end.
      await resync();
    },
    [apply, resync],
  );

  const stop = useCallback(async (): Promise<void> => {
    const jobId = jobIdRef.current;
    if (!jobId) {
      return;
    }
    setState({ phase: 'stopping', jobId });
    try {
      // A capture that ended as the stop was sent answers with its end.
      apply(await cancelJob(jobId));
    } catch {
      // The capture is still running, so the operator can try again.
      setState({ phase: 'running', jobId, stopFailed: true });
    }
  }, [apply]);

  return { state, start, stop };
}
