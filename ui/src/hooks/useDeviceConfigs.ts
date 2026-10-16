/**
 * Data hooks for the configuration backup reads (P-D1, P-D2, UI-SEED-25).
 *
 * Three reads chained by selection: the backup targets, one target's backup
 * attempts (newest first), and one attempt's change from the configuration
 * before it. Every read is operator+, so the page asks only for an operator.
 */

import { useEffect, useState } from 'react';
import { ApiError, api } from '../api/client';
import type {
  DeviceConfigBackup,
  DeviceConfigBackupList,
  DeviceConfigDiff,
  DeviceConfigTarget,
  DeviceConfigTargetList,
} from '../types/deviceConfigs';

const TARGETS = '/api/v1/device-config-targets';
const BACKUPS = '/api/v1/device-configs';

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export interface UseDeviceConfigTargetsResult {
  targets: DeviceConfigTarget[];
  loading: boolean;
  error: string | null;
}

export function useDeviceConfigTargets(enabled: boolean): UseDeviceConfigTargetsResult {
  const [targets, setTargets] = useState<DeviceConfigTarget[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    api
      .get<DeviceConfigTargetList>(TARGETS)
      .then((resp) => {
        if (!cancelled) {
          setTargets(resp.targets);
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
  }, [enabled]);

  return { targets, loading, error };
}

export interface UseDeviceConfigBackupsResult {
  /** Null while the target's backups are loading. */
  backups: DeviceConfigBackup[] | null;
  error: string | null;
}

export function useDeviceConfigBackups(targetId: string | null): UseDeviceConfigBackupsResult {
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
  }, [targetId]);

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
