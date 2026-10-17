/**
 * Data hooks for configuration backup (P-D1, P-D2, UI-SEED-25, UI-SEED-45).
 *
 * Three reads chained by selection: the backup targets, one target's backup
 * attempts (newest first), and one attempt's change from the configuration
 * before it. Every read and write is operator+, so the page asks only for an
 * operator. A run is a job: it is started here and followed on the shared job
 * event stream, and its end re-reads the targets and backups it changed.
 */

import { useCallback, useEffect, useState } from 'react';
import { ApiError, api } from '../api/client';
import { isTerminalJobState } from '../lib/jobsClient';
import type {
  DeviceConfigBackup,
  DeviceConfigBackupList,
  DeviceConfigDiff,
  DeviceConfigRunResult,
  DeviceConfigTarget,
  DeviceConfigTargetInput,
  DeviceConfigTargetList,
} from '../types/deviceConfigs';
import type { JobResponse } from '../types/generated/job-response';
import { useJobEvents } from './useJobEvents';

const TARGETS = '/api/v1/device-config-targets';
const BACKUPS = '/api/v1/device-configs';

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function targetPath(id: string): string {
  return `${TARGETS}/${encodeURIComponent(id)}`;
}

export function saveDeviceConfigTarget(
  input: DeviceConfigTargetInput,
  id?: string,
): Promise<DeviceConfigTarget> {
  return id
    ? api.put<DeviceConfigTarget>(targetPath(id), input)
    : api.post<DeviceConfigTarget>(TARGETS, input);
}

/** Deleting a target deletes its saved configurations with it. */
export function deleteDeviceConfigTarget(id: string): Promise<void> {
  return api.delete<void>(targetPath(id));
}

/** Pins the pending key; the fingerprint is the one the operator was shown. */
export function confirmDeviceConfigHostKey(id: string, fingerprint: string): Promise<void> {
  return api.put<void>(`${targetPath(id)}/host-key`, { fingerprint });
}

/** Forgets the pinned key, so the next backup holds the device's key for confirmation. */
export function forgetDeviceConfigHostKey(id: string): Promise<void> {
  return api.delete<void>(`${targetPath(id)}/host-key`);
}

export interface UseDeviceConfigTargetsResult {
  targets: DeviceConfigTarget[];
  loading: boolean;
  error: string | null;
  reload: () => void;
}

export function useDeviceConfigTargets(): UseDeviceConfigTargetsResult {
  const [targets, setTargets] = useState<DeviceConfigTarget[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [version, setVersion] = useState(0);
  const reload = useCallback((): void => setVersion((v) => v + 1), []);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api
      .get<DeviceConfigTargetList>(TARGETS)
      .then((resp) => {
        if (!cancelled) {
          setTargets(resp.targets);
          setError(null);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(message(err));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return (): void => {
      cancelled = true;
    };
  }, [version]);

  return { targets, loading, error, reload };
}

/** A run's progress: idle, running as a job, or finished with its counts. */
export type BackupRunState =
  | { kind: 'idle' }
  | { kind: 'running'; jobId: string }
  | { kind: 'finished'; result: DeviceConfigRunResult }
  | { kind: 'failed'; message: string };

export interface UseBackupRunResult {
  run: BackupRunState;
  /** Backs up the given targets, or every target when none are given. */
  start: (targetIds?: string[]) => Promise<void>;
}

function isRunResult(value: unknown): value is DeviceConfigRunResult {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as DeviceConfigRunResult).succeeded === 'number' &&
    typeof (value as DeviceConfigRunResult).failed === 'number'
  );
}

/** Starts backup runs and calls onFinished when the running one ends. */
export function useBackupRun(onFinished: () => void): UseBackupRunResult {
  const [run, setRun] = useState<BackupRunState>({ kind: 'idle' });
  const runningId = run.kind === 'running' ? run.jobId : null;

  useJobEvents((job: JobResponse) => {
    if (job.id !== runningId || !isTerminalJobState(job.state)) {
      return;
    }
    if (job.state === 'succeeded' && isRunResult(job.result)) {
      setRun({ kind: 'finished', result: job.result });
    } else {
      setRun({ kind: 'failed', message: job.error || job.state });
    }
    onFinished();
  });

  const start = async (targetIds?: string[]): Promise<void> => {
    try {
      const job = await api.post<JobResponse>(`${BACKUPS}/run`, targetIds ? { targetIds } : {});
      setRun({ kind: 'running', jobId: job.id });
    } catch (err: unknown) {
      setRun({ kind: 'failed', message: message(err) });
    }
  };

  return { run, start };
}

export interface UseDeviceConfigBackupsResult {
  /** Null while the target's backups are loading. */
  backups: DeviceConfigBackup[] | null;
  error: string | null;
}

/** Re-reads whenever version changes, so a finished run shows its attempts. */
export function useDeviceConfigBackups(
  targetId: string | null,
  version = 0,
): UseDeviceConfigBackupsResult {
  const [backups, setBackups] = useState<DeviceConfigBackup[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setBackups(null);
    setError(null);
    if (!targetId) {
      return;
    }
    let cancelled = false;
    api
      .get<DeviceConfigBackupList>(`${BACKUPS}?targetId=${encodeURIComponent(targetId)}`)
      .then((resp) => {
        if (!cancelled) {
          setBackups(resp.backups);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(message(err));
        }
      });
    return (): void => {
      cancelled = true;
    };
  }, [targetId, version]);

  return { backups, error };
}

/**
 * The diff read's three outcomes. The target's first configuration has none
 * before it, which the server answers with 404; that is "first", not an error.
 */
export type DeviceConfigDiffState =
  | { kind: 'loading' }
  | { kind: 'ready'; diff: DeviceConfigDiff }
  | { kind: 'first' }
  | { kind: 'error'; message: string };

export function useDeviceConfigDiff(backupId: string | null): DeviceConfigDiffState {
  const [state, setState] = useState<DeviceConfigDiffState>({ kind: 'loading' });

  useEffect(() => {
    setState({ kind: 'loading' });
    if (!backupId) {
      return;
    }
    let cancelled = false;
    api
      .get<DeviceConfigDiff>(`${BACKUPS}/${encodeURIComponent(backupId)}/diff`)
      .then((diff) => {
        if (!cancelled) {
          setState({ kind: 'ready', diff });
        }
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return;
        }
        setState(
          err instanceof ApiError && err.status === 404
            ? { kind: 'first' }
            : { kind: 'error', message: message(err) },
        );
      });
    return (): void => {
      cancelled = true;
    };
  }, [backupId]);

  return state;
}
