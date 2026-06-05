/**
 * jobsClient — typed client for the unified job runner spine (ADR-0005).
 *
 * The backend exposes long-running operations through one /api/v1/jobs
 * surface: POST submits a job of a given kind, GET snapshots it, DELETE
 * cancels it, and GET /jobs/events streams every state change (see
 * useJobEvents). These thin wrappers ride the shared CSRF-aware `api`
 * client so credentials, CSRF tokens, and 401-refresh are handled the
 * same way as every other call.
 *
 * The request params and job result are intentionally opaque (`unknown`
 * in the generated types): each kind owns its own params/result shape.
 * Callers narrow them at the use site.
 */

import { api } from '../api/client';
import type { CreateJobRequest } from '../types/generated/create-job-request';
import type { JobResponse } from '../types/generated/job-response';

/** JobState mirrors the platform/jobs runner lifecycle (ADR-0005). */
export type JobState = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled';

/** Base path for the jobs surface. */
const JOBS_BASE = '/api/v1/jobs';

/**
 * submitJob POSTs a new job of the given kind and returns the created
 * snapshot (state is usually `queued` or `running`). Pass an idempotency
 * key to make retries safe: the backend replays the original job for the
 * same key + body, and rejects a reused key with a different body (409).
 */
export async function submitJob(
  req: CreateJobRequest,
  idempotencyKey?: string,
): Promise<JobResponse> {
  const init: RequestInit | undefined = idempotencyKey
    ? { headers: { 'Idempotency-Key': idempotencyKey } }
    : undefined;
  return api.post<JobResponse>(JOBS_BASE, req, init);
}

/** getJob returns the current snapshot of a job by id. */
export async function getJob(id: string): Promise<JobResponse> {
  return api.get<JobResponse>(`${JOBS_BASE}/${encodeURIComponent(id)}`);
}

/**
 * cancelJob requests cancellation of a job. The backend acknowledges
 * asynchronously (202) and returns the snapshot at cancel time; the
 * terminal `cancelled` state arrives over the event stream.
 */
export async function cancelJob(id: string): Promise<JobResponse> {
  return api.delete<JobResponse>(`${JOBS_BASE}/${encodeURIComponent(id)}`);
}

/**
 * isTerminalJobState reports whether a job state is final — i.e. no
 * further events will arrive for it. Useful for stopping a watch.
 */
export function isTerminalJobState(state: string): boolean {
  return state === 'succeeded' || state === 'failed' || state === 'cancelled';
}
