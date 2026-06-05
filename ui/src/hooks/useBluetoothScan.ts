/**
 * useBluetoothScan — drive a Bluetooth scan through the unified jobs spine.
 *
 * Submits a `bluetooth-scan` job (ADR-0005), tracks it over the shared
 * /api/v1/jobs/events stream (useJobEvents), and exposes the scan lifecycle
 * plus the discovered devices. Unlike useEngineScan — which owns only the
 * lifecycle and lets devices load separately — the bluetooth-scan job returns
 * the full BluetoothScanResponse (devices + stats) as its job `result`, so
 * this hook surfaces that result directly once the job succeeds.
 *
 * The scan is synchronous server-side and reports no intermediate progress, so
 * percentComplete is 0 while running and 100 on completion (the card shows an
 * indeterminate "scanning…" state meanwhile).
 *
 * Bluetooth / BLE / RSSI / GATT / UUID are protocol nouns (Do-Not-Translate).
 */

import { useCallback, useRef, useState } from 'react';
import { cancelJob, submitJob } from '../lib/jobsClient';
import type {
  BluetoothDevice,
  BluetoothDiscoveryStats,
  BluetoothScanResponse,
} from '../types/generated/bluetooth-scan-response';
import type { JobResponse } from '../types/generated/job-response';
import { useJobEvents } from './useJobEvents';

/** BluetoothScanState is the card's view of the job lifecycle. */
export type BluetoothScanState = 'idle' | 'running' | 'complete' | 'failed' | 'canceled';

/** BluetoothScanStatus is the scan lifecycle the card renders. */
export interface BluetoothScanStatus {
  state: BluetoothScanState;
  jobId: string;
  /** 0 while running, 100 on completion (scan reports no intermediate progress). */
  percentComplete: number;
  error: string | null;
}

export interface UseBluetoothScanReturn {
  status: BluetoothScanStatus;
  running: boolean;
  /** The most recent successful scan result, or null before the first scan. */
  result: BluetoothScanResponse | null;
  /** Convenience accessor for result.devices (empty before the first scan). */
  devices: BluetoothDevice[];
  /** Convenience accessor for result.stats (null before the first scan). */
  stats: BluetoothDiscoveryStats | null;
  startScan: () => Promise<void>;
  cancelScan: () => Promise<void>;
}

const IDLE_STATUS: BluetoothScanStatus = {
  state: 'idle',
  jobId: '',
  percentComplete: 0,
  error: null,
};

/** toScanState maps a job lifecycle state to the card's coarser view. */
function toScanState(jobState: string): BluetoothScanState {
  switch (jobState) {
    case 'queued':
    case 'running':
      return 'running';
    case 'succeeded':
      return 'complete';
    case 'failed':
      return 'failed';
    case 'cancelled':
      return 'canceled';
    default:
      return 'idle';
  }
}

function statusFromJob(job: JobResponse): BluetoothScanStatus {
  const state = toScanState(job.state);
  return {
    state,
    jobId: job.id,
    // The scan emits no intermediate progress; show 100 only once complete.
    percentComplete: state === 'complete' ? 100 : 0,
    error: job.error ?? null,
  };
}

/**
 * isScanResponse narrows the opaque job result (typed `unknown`) to a
 * BluetoothScanResponse by checking for its `devices` array. The bluetooth-scan
 * job contract guarantees this shape on success; the guard keeps the cast safe.
 */
function isScanResponse(result: unknown): result is BluetoothScanResponse {
  return (
    typeof result === 'object' &&
    result !== null &&
    Array.isArray((result as { devices?: unknown }).devices)
  );
}

/**
 * useBluetoothScan submits and tracks one bluetooth-scan job at a time. The job
 * event stream is opened on mount; startScan/cancelScan drive the lifecycle and
 * the latest snapshot is exposed as `status`, with the discovered devices in
 * `result`/`devices`/`stats` once the job succeeds.
 */
export function useBluetoothScan(): UseBluetoothScanReturn {
  const [status, setStatus] = useState<BluetoothScanStatus>(IDLE_STATUS);
  const [result, setResult] = useState<BluetoothScanResponse | null>(null);
  const jobIdRef = useRef<string>('');

  // applyJob updates status from a job snapshot and captures the scan result
  // once it succeeds. Used both for live SSE updates and for the submit
  // response itself (a synchronous job may already be terminal on return).
  const applyJob = useCallback((job: JobResponse): void => {
    setStatus(statusFromJob(job));
    if (job.state === 'succeeded' && isScanResponse(job.result)) {
      setResult(job.result);
    }
  }, []);

  // Apply events for OUR job only; the stream multiplexes every job.
  useJobEvents(
    useCallback(
      (job: JobResponse) => {
        if (job.id !== jobIdRef.current) {
          return;
        }
        applyJob(job);
      },
      [applyJob],
    ),
  );

  const startScan = useCallback(async (): Promise<void> => {
    setStatus({ state: 'running', jobId: '', percentComplete: 0, error: null });
    try {
      // bluetooth-scan takes no params; the scanner is injected server-side.
      const job = await submitJob({ kind: 'bluetooth-scan' });
      jobIdRef.current = job.id;
      applyJob(job);
    } catch (err) {
      jobIdRef.current = '';
      setStatus({
        state: 'failed',
        jobId: '',
        percentComplete: 0,
        error: err instanceof Error ? err.message : 'Failed to start scan',
      });
      throw err;
    }
  }, []);

  const cancelScan = useCallback(async (): Promise<void> => {
    const id = jobIdRef.current;
    if (!id) {
      return;
    }
    try {
      await cancelJob(id);
      setStatus((prev) => ({ ...prev, state: 'canceled' }));
    } catch (err) {
      setStatus((prev) => ({
        ...prev,
        error: err instanceof Error ? err.message : 'Failed to cancel scan',
      }));
      throw err;
    }
  }, []);

  return {
    status,
    running: status.state === 'running',
    result,
    devices: result?.devices ?? [],
    stats: result?.stats ?? null,
    startScan,
    cancelScan,
  };
}
