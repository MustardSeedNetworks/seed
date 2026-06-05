import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { BluetoothScanResponse } from '../types/generated/bluetooth-scan-response';
import type { JobResponse } from '../types/generated/job-response';
import { useBluetoothScan } from './useBluetoothScan';

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

const queuedJob: JobResponse = { id: 'bt-1', kind: 'bluetooth-scan', state: 'queued', progress: 0 };

const scanResult: BluetoothScanResponse = {
  devices: [
    {
      id: 'dev-1',
      address: 'AA:BB:CC:DD:EE:FF',
      name: 'Pixel Buds',
      alias: '',
      vendor: 'Google',
      isConnected: false,
      type: 'ble',
      deviceClass: '',
      appearance: 0,
      rssi: -52,
      txPower: 0,
      estDistanceM: 1.2,
      isConnectable: true,
      companyName: 'Google',
      serviceNames: ['Battery'],
      isAuthorized: false,
      isTrusted: false,
      isPaired: false,
      isBlocked: false,
      firstSeen: '2026-06-05T00:00:00Z',
      lastSeen: '2026-06-05T00:00:00Z',
    },
  ],
  adapterName: 'hci0',
  scanType: 'dual',
  scanTime: '2026-06-05T00:00:00Z',
  scanDurationMs: 5000,
  stats: {
    totalDevices: 1,
    classicDevices: 0,
    bleDevices: 1,
    dualDevices: 0,
    connectedDevices: 0,
    authorizedCount: 0,
    unauthorizedCount: 1,
    devicesByClass: {},
    vendorBreakdown: { Google: 1 },
    lastScanTime: '2026-06-05T00:00:00Z',
  },
};

function emit(job: JobResponse): void {
  act(() => {
    jobEventCb?.(job);
  });
}

describe('useBluetoothScan', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    jobEventCb = null;
  });

  it('submits a bluetooth-scan job with no params', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useBluetoothScan());

    await act(async () => {
      await result.current.startScan();
    });

    expect(submitJob).toHaveBeenCalledTimes(1);
    const [req] = vi.mocked(submitJob).mock.calls[0];
    expect(req.kind).toBe('bluetooth-scan');
    expect(req.params).toBeUndefined();
    expect(result.current.running).toBe(true);
    expect(result.current.status.jobId).toBe('bt-1');
  });

  it('captures devices + stats from the succeeded job result', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useBluetoothScan());
    await act(async () => {
      await result.current.startScan();
    });

    expect(result.current.devices).toEqual([]); // empty before completion

    emit({ ...queuedJob, state: 'succeeded', progress: 1, result: scanResult });

    expect(result.current.status.state).toBe('complete');
    expect(result.current.status.percentComplete).toBe(100);
    expect(result.current.running).toBe(false);
    expect(result.current.devices).toHaveLength(1);
    expect(result.current.devices[0].companyName).toBe('Google');
    expect(result.current.devices[0].serviceNames).toEqual(['Battery']);
    expect(result.current.stats?.bleDevices).toBe(1);
  });

  it('shows 0% while running (scan reports no intermediate progress)', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useBluetoothScan());
    await act(async () => {
      await result.current.startScan();
    });

    emit({ ...queuedJob, state: 'running', progress: 0 });
    expect(result.current.status.state).toBe('running');
    expect(result.current.status.percentComplete).toBe(0);
  });

  it('ignores a malformed result without crashing', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useBluetoothScan());
    await act(async () => {
      await result.current.startScan();
    });

    emit({ ...queuedJob, state: 'succeeded', progress: 1, result: { unexpected: true } });

    expect(result.current.status.state).toBe('complete');
    expect(result.current.result).toBeNull(); // guard rejected the bad shape
    expect(result.current.devices).toEqual([]);
  });

  it('ignores events for a different job', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    const { result } = renderHook(() => useBluetoothScan());
    await act(async () => {
      await result.current.startScan();
    });

    emit({
      id: 'other-job',
      kind: 'bluetooth-scan',
      state: 'succeeded',
      progress: 1,
      result: scanResult,
    });

    expect(result.current.status.state).toBe('running'); // unchanged
    expect(result.current.devices).toEqual([]);
  });

  it('surfaces a failed submit', async () => {
    vi.mocked(submitJob).mockRejectedValueOnce(new Error('adapter busy'));
    const { result } = renderHook(() => useBluetoothScan());

    await expect(result.current.startScan()).rejects.toThrow('adapter busy');
    await waitFor(() => {
      expect(result.current.status.state).toBe('failed');
      expect(result.current.status.error).toBe('adapter busy');
    });
  });

  it('cancels the running job', async () => {
    vi.mocked(submitJob).mockResolvedValueOnce(queuedJob);
    vi.mocked(cancelJob).mockResolvedValueOnce({ ...queuedJob, state: 'cancelled' });
    const { result } = renderHook(() => useBluetoothScan());
    await act(async () => {
      await result.current.startScan();
    });
    await act(async () => {
      await result.current.cancelScan();
    });

    expect(cancelJob).toHaveBeenCalledWith('bt-1');
    expect(result.current.status.state).toBe('canceled');
  });

  it('cancel is a no-op when no scan is running', async () => {
    const { result } = renderHook(() => useBluetoothScan());
    await act(async () => {
      await result.current.cancelScan();
    });
    expect(cancelJob).not.toHaveBeenCalled();
  });
});
