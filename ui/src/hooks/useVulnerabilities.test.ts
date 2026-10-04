import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { JobResponse } from '../types/generated/job-response';
import { useVulnerabilities } from './useVulnerabilities';

// Mock the jobs client and capture the useJobEvents callback so the test can
// deliver job frames.
vi.mock('../lib/jobsClient', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/jobsClient')>()),
  submitJob: vi.fn(),
  getJob: vi.fn(),
}));

let jobEventCb: ((job: JobResponse) => void) | null = null;
vi.mock('./useJobEvents', () => ({
  useJobEvents: (cb: (job: JobResponse) => void) => {
    jobEventCb = cb;
    return { status: 'open' };
  },
}));

import { getJob, submitJob } from '../lib/jobsClient';
import { must } from '../test/must';

const queued: JobResponse = { id: 'vs-1', kind: 'vuln-scan', state: 'queued', progress: 0 };
const succeeded: JobResponse = { ...queued, state: 'succeeded', progress: 1 };

beforeEach(() => {
  vi.mocked(submitJob).mockReset().mockResolvedValue(queued);
  vi.mocked(getJob).mockReset().mockResolvedValue(queued);
  jobEventCb = null;
});

describe('useVulnerabilities triggerScan', () => {
  it('submits one vuln-scan job naming the device', async () => {
    vi.mocked(submitJob).mockResolvedValue(succeeded);
    const { result } = renderHook(() => useVulnerabilities());

    await act(() => result.current.triggerScan(' 192.0.2.10 '));

    expect(submitJob).toHaveBeenCalledWith({ kind: 'vuln-scan', params: { ip: '192.0.2.10' } });
  });

  it('scans every device when no address is given', async () => {
    vi.mocked(submitJob).mockResolvedValue(succeeded);
    const { result } = renderHook(() => useVulnerabilities());

    await act(() => result.current.triggerScan());

    expect(submitJob).toHaveBeenCalledWith({ kind: 'vuln-scan', params: {} });
  });

  it('resolves only when the job stream reports the scan finished', async () => {
    const { result } = renderHook(() => useVulnerabilities());

    let done: boolean | undefined;
    let scan: Promise<void> = Promise.resolve();
    await act(async () => {
      scan = result.current.triggerScan('192.0.2.10').then((ok) => {
        done = ok;
      });
      await vi.waitFor(() => expect(getJob).toHaveBeenCalledWith(queued.id));
    });
    expect(done).toBeUndefined();
    expect(result.current.isScanning).toBe(true);

    await act(async () => {
      must(jobEventCb)({ ...queued, state: 'running' });
      await Promise.resolve();
    });
    expect(done).toBeUndefined();

    await act(async () => {
      must(jobEventCb)(succeeded);
      await scan;
    });
    expect(done).toBe(true);
    expect(result.current.isScanning).toBe(false);
  });

  it('catches a scan that finished before the stream could report it', async () => {
    vi.mocked(getJob).mockResolvedValue(succeeded);
    const { result } = renderHook(() => useVulnerabilities());

    await expect(act(() => result.current.triggerScan('192.0.2.10'))).resolves.toBe(true);
  });

  it('reports a failed scan', async () => {
    vi.mocked(submitJob).mockResolvedValue({ ...queued, state: 'failed', error: 'no device' });
    const { result } = renderHook(() => useVulnerabilities());

    await expect(act(() => result.current.triggerScan('192.0.2.10'))).resolves.toBe(false);
    expect(result.current.scanError).toBe('no device');
  });

  it('refuses an address before submitting anything', async () => {
    const { result } = renderHook(() => useVulnerabilities());

    await expect(act(() => result.current.triggerScan('not-an-ip'))).resolves.toBe(false);
    expect(submitJob).not.toHaveBeenCalled();
  });
});
