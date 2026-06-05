/**
 * useEngineScan — drive a discovery scan through the unified jobs spine.
 *
 * This is the Phase 7 S3 replacement for usePipelineStatus: instead of the
 * legacy /api/v1/security/pipeline/* endpoints, it submits an `engine-scan`
 * job (ADR-0005/0007), tracks it over the shared /api/v1/jobs/events stream
 * (useJobEvents), and exposes a small status the discovery card renders.
 *
 * Devices are NOT returned here — they continue to load via
 * useDiscoveredDevices (GET /api/v1/security/devices), unchanged. This hook
 * owns only the scan lifecycle (start / progress / cancel).
 *
 * Progress is the job's cumulative fraction (the engine reports it per phase,
 * S4.2). The phase *name* rides the engine event bus, not the job stream, so
 * it is intentionally not surfaced here (see SEED_S4_FOLD_PLAN.md S3 fork).
 */

import { useCallback, useRef, useState } from 'react';
import { cancelJob, submitJob } from '../lib/jobsClient';
import type { EngineScanRequest } from '../types/generated/engine-scan-request';
import type { JobResponse } from '../types/generated/job-response';
import { useJobEvents } from './useJobEvents';

/** EngineScanState is the discovery-card view of the job lifecycle. */
export type EngineScanState = 'idle' | 'running' | 'complete' | 'failed' | 'canceled';

/** EngineScanStatus is the scan lifecycle the discovery card renders. */
export interface EngineScanStatus {
  state: EngineScanState;
  jobId: string;
  /** Cumulative progress, 0..100. */
  percentComplete: number;
  error: string | null;
}

export interface UseEngineScanReturn {
  status: EngineScanStatus;
  running: boolean;
  startScan: (overrides?: Partial<EngineScanRequest>) => Promise<void>;
  cancelScan: () => Promise<void>;
}

/**
 * DEFAULT_SCAN_PARAMS mirrors the discovery card's prior pipeline config:
 * name resolution + service discovery (port scan + profiling) at quick
 * intensity, no vuln assessment, fresh discovery across all transports.
 */
const DEFAULT_SCAN_PARAMS: EngineScanRequest = {
  scanType: 'full',
  includeWired: true,
  includeWifi: true,
  includeBluetooth: true,
  includeSnmp: false,
  includePortScan: true,
  includeProfiling: true,
  includeNameRes: true,
  includeVulnScan: false,
  freshWiredScan: true,
  freshWifiScan: true,
  freshBluetoothScan: true,
  portScanIntensity: 'quick',
};

const IDLE_STATUS: EngineScanStatus = {
  state: 'idle',
  jobId: '',
  percentComplete: 0,
  error: null,
};

/** toScanState maps a job lifecycle state to the card's coarser view. */
function toScanState(jobState: string): EngineScanState {
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

/** pct clamps a job progress fraction (0..1) to an integer percentage. */
function pct(fraction: number): number {
  return Math.round(Math.min(Math.max(fraction, 0), 1) * 100);
}

function statusFromJob(job: JobResponse): EngineScanStatus {
  return {
    state: toScanState(job.state),
    jobId: job.id,
    percentComplete: pct(job.progress),
    error: job.error ?? null,
  };
}

/**
 * useEngineScan submits and tracks one discovery engine-scan job at a time.
 * The job event stream is opened on mount; startScan/cancelScan drive the
 * lifecycle and the latest snapshot is exposed as `status`.
 */
export function useEngineScan(): UseEngineScanReturn {
  const [status, setStatus] = useState<EngineScanStatus>(IDLE_STATUS);
  const jobIdRef = useRef<string>('');

  // Apply events for OUR job only; the stream multiplexes every job.
  useJobEvents(
    useCallback((job: JobResponse) => {
      if (job.id !== jobIdRef.current) {
        return;
      }
      setStatus(statusFromJob(job));
    }, []),
  );

  const startScan = useCallback(async (overrides?: Partial<EngineScanRequest>): Promise<void> => {
    const params: EngineScanRequest = { ...DEFAULT_SCAN_PARAMS, ...overrides };
    setStatus({ state: 'running', jobId: '', percentComplete: 0, error: null });
    try {
      const job = await submitJob({ kind: 'engine-scan', params });
      jobIdRef.current = job.id;
      setStatus(statusFromJob(job));
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

  return { status, running: status.state === 'running', startScan, cancelScan };
}
