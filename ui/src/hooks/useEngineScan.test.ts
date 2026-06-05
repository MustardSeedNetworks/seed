import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { JobResponse } from '../types/generated/job-response';
import { useEngineScan } from './useEngineScan';

// Mock the jobs client (submit/cancel) and capture the useJobEvents callback so
// the test can drive job lifecycle events.
vi.mock('../lib/jobsClient', () => ({
  submitJob: vi.fn(),
  cancelJob: vi.fn(),
}));

let jobEventCb: ((job: JobResponse) => void) | null = null;
vi.mock('./useJobEvents', () => ({
  useJobEvents: (cb: (job: JobResponse) => void) => {
    jobEventCb = cb;
    return { status: 'open' };
  },
}));

import { cancelJob, submitJob } from '../lib/jobsClient';

const queuedJob: JobResponse = { id: 'job-1', kind: 'engine-scan', state: 'queued', progress: 0 };

function emit(job: JobResponse): void {
  act(() => {
    jobEventCb?.(job);
  });
}

describe('useEngineScan', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    jobEventCb = null;
  });

  it('submits an engine-scan job with the card default params', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useEngineScan());

    await act(async () => {
      await result.current.startScan();
    });

    expect(submitJob).toHaveBeenCalledTimes(1);
    const [req] = vi.mocked(submitJob).mock.calls[0];
    expect(req.kind).toBe('engine-scan');
    const params = req.params as Record<string, unknown>;
    expect(params.includeNameRes).toBe(true);
    expect(params.includeProfiling).toBe(true);
    expect(params.includeVulnScan).toBe(false);
    expect(params.portScanIntensity).toBe('quick');
    expect(result.current.running).toBe(true);
    expect(result.current.status.jobId).toBe('job-1');
  });

  it('updates progress + state from events for its own job', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useEngineScan());
    await act(async () => {
      await result.current.startScan();
    });

    emit({ ...queuedJob, state: 'running', progress: 0.4 });
    expect(result.current.status.percentComplete).toBe(40);
    expect(result.current.status.state).toBe('running');

    emit({ ...queuedJob, state: 'succeeded', progress: 1 });
    expect(result.current.status.state).toBe('complete');
    expect(result.current.status.percentComplete).toBe(100);
    expect(result.current.running).toBe(false);
  });

  it('ignores events for a different job', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useEngineScan());
    await act(async () => {
      await result.current.startScan();
    });

    emit({ id: 'other-job', kind: 'engine-scan', state: 'succeeded', progress: 1 });

    expect(result.current.status.state).toBe('running'); // unchanged
    expect(result.current.status.jobId).toBe('job-1');
  });

  it('surfaces a failed submit', async () => {
    vi.mocked(submitJob).mockRejectedValueOnce(new Error('at capacity'));
    const { result } = renderHook(() => useEngineScan());

    await expect(result.current.startScan()).rejects.toThrow('at capacity');
    await waitFor(() => {
      expect(result.current.status.state).toBe('failed');
      expect(result.current.status.error).toBe('at capacity');
    });
  });

  it('cancels the running job', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    vi.mocked(cancelJob).mockResolvedValueOnce({ ...queuedJob, state: 'cancelled' });
    const { result } = renderHook(() => useEngineScan());
    await act(async () => {
      await result.current.startScan();
    });

    await act(async () => {
      await result.current.cancelScan();
    });

    expect(cancelJob).toHaveBeenCalledWith('job-1');
    expect(result.current.status.state).toBe('canceled');
  });

  it('cancel is a no-op when no scan is running', async () => {
    const { result } = renderHook(() => useEngineScan());
    await act(async () => {
      await result.current.cancelScan();
    });
    expect(cancelJob).not.toHaveBeenCalled();
  });
});
