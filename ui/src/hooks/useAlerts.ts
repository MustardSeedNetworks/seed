/**
 * Alerts hook covering list + acknowledge + resolve. Refreshes the
 * list after every action so the table reflects server state
 * (acknowledgedAt timestamp etc.) instead of an optimistic stale
 * row. The endpoint is operator-paced so the extra round-trip is
 * cheap.
 */

import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Alert, AlertActionResponse, AlertsFilter, AlertsListResponse } from '../types/alerts';

const ENDPOINT = '/api/v1/alerts';

function buildQuery(filter: AlertsFilter): string {
  const params = new URLSearchParams();
  if (filter.severity) params.set('severity', filter.severity);
  if (filter.unacknowledgedOnly) params.set('unacknowledged_only', 'true');
  if (filter.unresolvedOnly) params.set('unresolved_only', 'true');
  const q = params.toString();
  return q ? `?${q}` : '';
}

export interface UseAlertsResult {
  alerts: Alert[];
  loading: boolean;
  error: string | null;
  filter: AlertsFilter;
  setFilter: (next: AlertsFilter) => void;
  refresh: () => Promise<void>;
  acknowledge: (id: number) => Promise<void>;
  resolve: (id: number) => Promise<void>;
}

export function useAlerts(initialFilter?: Partial<AlertsFilter>): UseAlertsResult {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<AlertsFilter>({
    severity: '',
    unacknowledgedOnly: false,
    unresolvedOnly: false,
    ...initialFilter,
  });

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.get<AlertsListResponse>(`${ENDPOINT}${buildQuery(filter)}`);
      setAlerts(resp.alerts ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load alerts');
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const acknowledge = useCallback(
    async (id: number): Promise<void> => {
      await api.post<AlertActionResponse>(`${ENDPOINT}/${id}/acknowledge`);
      await refresh();
    },
    [refresh],
  );

  const resolve = useCallback(
    async (id: number): Promise<void> => {
      await api.post<AlertActionResponse>(`${ENDPOINT}/${id}/resolve`);
      await refresh();
    },
    [refresh],
  );

  return { alerts, loading, error, filter, setFilter, refresh, acknowledge, resolve };
}
