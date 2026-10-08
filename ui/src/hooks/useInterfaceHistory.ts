/**
 * Data hook for GET /api/v1/topology/interfaces/history: one interface's
 * rates over a recent window (UI-SEED-21).
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { InterfaceHistoryResponse } from '../types/generated/interface-history-response';

export const HISTORY_RANGES = ['1h', '24h', '7d'] as const;
export type HistoryRange = (typeof HISTORY_RANGES)[number];

export interface UseInterfaceHistoryResult {
  history: InterfaceHistoryResponse | null;
  loading: boolean;
  error: string | null;
}

export function useInterfaceHistory(
  targetId: string,
  ifIndex: number,
  range: HistoryRange,
): UseInterfaceHistoryResult {
  const [history, setHistory] = useState<InterfaceHistoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const query = new URLSearchParams({ target: targetId, ifIndex: String(ifIndex), range });
    setLoading(true);
    setError(null);
    api
      .get<InterfaceHistoryResponse>(`/api/v1/topology/interfaces/history?${query.toString()}`)
      .then((resp) => {
        if (!cancelled) {
          setHistory(resp);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
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
  }, [targetId, ifIndex, range]);

  return { history, loading, error };
}
