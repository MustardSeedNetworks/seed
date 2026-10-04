/**
 * useReports owns the reports API calls for ReportsCard.
 *
 * Kept apart from the card so the card stays a pure render of its props, and so
 * the mapping from the wire shape to those props has somewhere to be tested.
 */
import { useEffect, useState } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { ReportInfo, ReportsResponse } from '../types/generated/reports-response';

const reportsEndpoint = '/api/v1/reports';

/** Report formats the generator implements. xlsx and md are defined but not. */
export type ReportFormat = 'pdf' | 'html' | 'csv' | 'json';

export interface UseReportsResult {
  reports: ReportInfo[];
  loading: boolean;
  error: string | null;
  generating: boolean;
  refresh: () => Promise<void>;
  generate: (type: string, format: ReportFormat) => Promise<void>;
  remove: (id: string) => Promise<void>;
}

function isReportsResponse(value: unknown): value is ReportsResponse {
  return (
    typeof value === 'object' &&
    value !== null &&
    Array.isArray((value as { reports?: unknown }).reports)
  );
}

async function fetchReports(): Promise<{ reports: ReportInfo[]; error: string | null }> {
  const res = await fetch(reportsEndpoint, { credentials: 'include' });
  if (!res.ok) {
    // 402 is the licence gate and 401 the auth boundary; neither is a
    // failure worth a red card, but an empty list would be a lie.
    return { reports: [], error: `reports request failed (${res.status})` };
  }
  const body: unknown = await res.json();
  return { reports: isReportsResponse(body) ? body.reports : [], error: null };
}

export function useReports(): UseReportsResult {
  const [reports, setReports] = useState<ReportInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);

  const refresh = async () => {
    setLoading(true);
    await fetchReports()
      .then((loaded) => {
        setReports(loaded.reports);
        setError(loaded.error);
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.EXPORT, 'Failed to load reports', err);
        setError(err instanceof Error ? err.message : 'Failed to load reports');
        setReports([]);
      });
    setLoading(false);
  };

  const generate = async (type: string, format: ReportFormat) => {
    setGenerating(true);
    try {
      // Through the api client, not a raw fetch: the client attaches the
      // X-CSRF-Token this route requires. A raw fetch omits it, and the
      // middleware answers 403 "CSRF token required" — so Generate Report
      // failed every time it was pressed.
      await api.post(`${reportsEndpoint}/generate`, { type, format });
      // 202: the record exists, the file does not yet. Re-read rather than
      // trusting the snapshot, so the row shows its real current status.
      await refresh();
    } catch (err) {
      logger.error(LogComponents.EXPORT, 'Failed to generate report', err);
      setError(err instanceof Error ? err.message : 'Failed to generate report');
    }
    setGenerating(false);
  };

  const remove = async (id: string) => {
    try {
      await api.delete(`${reportsEndpoint}/${encodeURIComponent(id)}`);
      await refresh();
    } catch (err) {
      logger.error(LogComponents.EXPORT, 'Failed to delete report', err);
      setError(err instanceof Error ? err.message : 'Failed to delete report');
    }
  };

  useEffect(() => {
    refresh().catch(() => undefined);
  }, [refresh]);

  return { reports, loading, error, generating, refresh, generate, remove };
}
