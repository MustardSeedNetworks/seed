/**
 * useGuestNetworkAudit
 *
 * Drives the on-demand Guest Network isolation audit (#397).
 *
 * The hook:
 * - Loads the configured target list from /api/v1/shell/guest-audit/settings.
 * - Exposes saveSettings to persist target/port edits.
 * - Exposes runAudit which fires /api/v1/shell/guest-audit/run and stores
 *   the most recent report, including the high-level isolationFailed flag
 *   that the UI surfaces as a critical security alert.
 */

import { useCallback, useEffect, useState } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';

export interface GuestAuditTarget {
  ip: string;
  label?: string;
}

export interface GuestAuditSettings {
  enabled: boolean;
  targets: GuestAuditTarget[];
  ports?: number[];
}

export interface GuestAuditPortResult {
  port: number;
  open: boolean;
  error?: string;
}

export interface GuestAuditTargetResult {
  target: GuestAuditTarget;
  reachable: boolean;
  pingResponded: boolean;
  openPorts: number[];
  portResults: GuestAuditPortResult[];
  durationSeconds: number;
}

export interface GuestAuditReport {
  startedAt: string;
  completedAt: string;
  isolationFailed: boolean;
  reachableTargets: number;
  totalTargets: number;
  results: GuestAuditTargetResult[];
}

interface UseGuestNetworkAuditResult {
  settings: GuestAuditSettings;
  setSettings: (next: GuestAuditSettings) => void;
  saveSettings: () => Promise<void>;
  runAudit: () => Promise<void>;
  report: GuestAuditReport | null;
  loading: boolean;
  running: boolean;
  error: string | null;
}

const DEFAULT_SETTINGS: GuestAuditSettings = {
  enabled: false,
  targets: [],
};

export function useGuestNetworkAudit(): UseGuestNetworkAuditResult {
  const [settings, setSettings] = useState<GuestAuditSettings>(DEFAULT_SETTINGS);
  const [report, setReport] = useState<GuestAuditReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Initial load
  useEffect(() => {
    setLoading(true);
    api
      .get<GuestAuditSettings>('/api/v1/shell/guest-audit/settings')
      .then((data) => {
        setSettings({
          enabled: data.enabled ?? false,
          targets: data.targets ?? [],
          ports: data.ports,
        });
      })
      .catch((err: unknown) => {
        logger.warn(LogComponents.CONFIG, 'Failed to load guest-audit settings', err);
      })
      .finally(() => setLoading(false));
  }, []);

  const saveSettings = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      await api.put('/api/v1/shell/guest-audit/settings', settings);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to save guest-audit settings';
      setError(msg);
      logger.error(LogComponents.CONFIG, 'Failed to save guest-audit settings', err);
      throw err;
    }
  }, [settings]);

  const runAudit = useCallback(async (): Promise<void> => {
    setRunning(true);
    setError(null);
    try {
      const result = await api.post<GuestAuditReport>('/api/v1/shell/guest-audit/run', {});
      setReport(result);
      if (result.isolationFailed) {
        logger.warn(LogComponents.VULN, 'Guest network isolation FAILED', {
          reachableTargets: result.reachableTargets,
          totalTargets: result.totalTargets,
        });
      } else {
        logger.info(LogComponents.VULN, 'Guest network isolation verified', {
          totalTargets: result.totalTargets,
        });
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to run guest-network audit';
      setError(msg);
      logger.error(LogComponents.VULN, 'Failed to run guest-network audit', err);
    } finally {
      setRunning(false);
    }
  }, []);

  return {
    settings,
    setSettings,
    saveSettings,
    runAudit,
    report,
    loading,
    running,
    error,
  };
}
