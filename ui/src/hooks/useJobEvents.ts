/**
 * useJobEvents — subscribe to the unified job event stream.
 *
 * The backend multiplexes every job lifecycle transition (queued →
 * running → succeeded/failed/cancelled) onto one SSE stream at
 * /api/v1/jobs/events as named `job` frames whose data is a JobResponse
 * (handlers_jobs.go). This hook opens that stream once and invokes
 * `onJob` for each frame; the browser's EventSource handles reconnection.
 *
 * It is the read side of the jobs spine: callers submit work with
 * jobsClient.submitJob and observe progress/completion here, rather than
 * polling per-job. A single stream serves all in-flight jobs, so mount
 * one subscription and route by job id at the callback.
 */

import { useEffect, useRef, useState } from 'react';
import { LogComponents, logger } from '../lib/logger';
import type { JobResponse } from '../types/generated/job-response';

/** Path of the multiplexed job event stream. */
const JOBS_EVENTS_URL = '/api/v1/jobs/events';

/** Connection status of the job event stream. */
export type JobEventsStatus = 'connecting' | 'open' | 'closed';

/**
 * useJobEvents opens the job SSE stream and calls `onJob` for every
 * state-change frame. Returns the live connection status. The callback
 * is held in a ref so changing it does not tear down the stream.
 */
export function useJobEvents(onJob: (job: JobResponse) => void): { status: JobEventsStatus } {
  const [status, setStatus] = useState<JobEventsStatus>('connecting');

  // Hold the callback in a ref so a new function identity each render
  // does not re-open the EventSource (the effect deliberately has no deps).
  const onJobRef = useRef(onJob);
  useEffect(() => {
    onJobRef.current = onJob;
  }, [onJob]);

  useEffect(() => {
    // EventSource does not accept relative URLs in all browsers; build an
    // absolute one the same way useSse does.
    const fullUrl = `${window.location.protocol}//${window.location.host}${JOBS_EVENTS_URL}`;
    const source = new EventSource(fullUrl, { withCredentials: true });
    setStatus('connecting');

    source.addEventListener('open', () => setStatus('open'));

    source.addEventListener('job', (event: MessageEvent<string>) => {
      try {
        const job = JSON.parse(event.data) as JobResponse;
        onJobRef.current(job);
      } catch (error) {
        logger.error(LogComponents.SSE, 'Failed to parse job event', error, {
          data: event.data,
        });
      }
    });

    source.addEventListener('error', () => {
      // EventSource auto-reconnects unless it has hard-closed. Reflect the
      // state so callers can surface a connection indicator.
      setStatus(source.readyState === EventSource.CLOSED ? 'closed' : 'connecting');
    });

    return () => source.close();
  }, []);

  return { status };
}
