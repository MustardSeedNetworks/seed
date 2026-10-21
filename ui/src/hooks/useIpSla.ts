/**
 * Data hooks for the /api/v1/ip-sla surface (P-D8, UI-SEED-48): the Cisco
 * IP-SLA operations the polling targets run, failing ones first, and one
 * operation's trend over a recent window.
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { HistoryRange } from '../components/ui/TrendChart';
import type { IPSLAHistoryResponse } from '../types/generated/ip-sla-history-response';
import type { IPSLAOperationsResponse } from '../types/generated/ip-sla-operations-response';

export interface UseIpSlaResult {
  data: IPSLAOperationsResponse | null;
  error: string | null;
}

export function useIpSla(): UseIpSlaResult {
  const [data, setData] = useState<IPSLAOperationsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .get<IPSLAOperationsResponse>('/api/v1/ip-sla/operations')
      .then(setData)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  return { data, error };
}

export interface UseIpSlaHistoryResult {
  history: IPSLAHistoryResponse | null;
  loading: boolean;
  error: string | null;
}

export function useIpSlaHistory(
  targetId: string,
  index: number,
  range: HistoryRange,
): UseIpSlaHistoryResult {
  const [history, setHistory] = useState<IPSLAHistoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const query = new URLSearchParams({ target: targetId, index: String(index), range });
    setLoading(true);
    setError(null);
    api
      .get<IPSLAHistoryResponse>(`/api/v1/ip-sla/operations/history?${query.toString()}`)
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
  }, [targetId, index, range]);

  return { history, loading, error };
}
