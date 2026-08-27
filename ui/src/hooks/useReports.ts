/**
 * useReports owns the reports API calls for ReportsCard.
 *
 * Kept apart from the card so the card stays a pure render of its props, and so
 * the mapping from the wire shape to those props has somewhere to be tested.
 */
import { useCallback, useEffect, useState } from 'react';
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

export function useReports(): UseReportsResult {
  const [reports, setReports] = useState<ReportInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(reportsEndpoint, { credentials: 'include' });
      if (!res.ok) {
        // 402 is the licence gate and 401 the auth boundary; neither is a
        // failure worth a red card, but an empty list would be a lie.
        setError(`reports request failed (${res.status})`);
        setReports([]);
        return;
      }
      const body: unknown = await res.json();
      setReports(isReportsResponse(body) ? body.reports : []);
      setError(null);
    } catch (err) {
      logger.error(LogComponents.EXPORT, 'Failed to load reports', err);
      setError(err instanceof Error ? err.message : 'Failed to load reports');
      setReports([]);
    } finally {
      setLoading(false);
    }
  }, []);

  const generate = useCallback(
    async (type: string, format: ReportFormat) => {
      setGenerating(true);
      try {
        const res = await fetch(`${reportsEndpoint}/generate`, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type, format }),
        });
        if (!res.ok) {
          setError(`report generation failed (${res.status})`);
          return;
        }
        // 202: the record exists, the file does not yet. Re-read rather than
        // trusting the snapshot, so the row shows its real current status.
        await refresh();
      } catch (err) {
        logger.error(LogComponents.EXPORT, 'Failed to generate report', err);
        setError(err instanceof Error ? err.message : 'Failed to generate report');
      } finally {
        setGenerating(false);
      }
    },
    [refresh],
  );

  const remove = useCallback(
    async (id: string) => {
      try {
        const res = await fetch(`${reportsEndpoint}/${encodeURIComponent(id)}`, {
          method: 'DELETE',
          credentials: 'include',
        });
        if (!res.ok) {
          setError(`report delete failed (${res.status})`);
          return;
        }
        await refresh();
      } catch (err) {
        logger.error(LogComponents.EXPORT, 'Failed to delete report', err);
        setError(err instanceof Error ? err.message : 'Failed to delete report');
      }
    },
    [refresh],
  );

  useEffect(() => {
    refresh().catch(() => undefined);
  }, [refresh]);

  return { reports, loading, error, generating, refresh, generate, remove };
}
