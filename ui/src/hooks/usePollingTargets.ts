/**
 * Data hook for the /api/v1/polling-targets surface.
 *
 * Exposes the read-list + the three CRUD mutations as a single
 * object so the page component doesn't have to weave together
 * useEffect, fetch state, and refresh-on-mutate logic itself.
 *
 * The hook is intentionally framework-light: no react-query,
 * no swr. The endpoint count is small and the page is operator-
 * facing (not a hot path), so the cost of bringing in a cache
 * library outweighs the benefit. If a second consumer of these
 * endpoints arrives, lift state into a context first.
 */

import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type {
  PollingTarget,
  PollingTargetInput,
  PollingTargetsListResponse,
} from '../types/polling';

const ENDPOINT = '/api/v1/polling-targets';

export interface UsePollingTargetsResult {
  targets: PollingTarget[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  create: (input: PollingTargetInput) => Promise<PollingTarget>;
  update: (id: string, input: PollingTargetInput) => Promise<PollingTarget>;
  remove: (id: string) => Promise<void>;
}

export function usePollingTargets(): UsePollingTargetsResult {
  const [targets, setTargets] = useState<PollingTarget[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.get<PollingTargetsListResponse>(ENDPOINT);
      setTargets(resp.targets ?? []);
    } catch (err) {
      // The api client throws Error on non-2xx; surface the message
      // verbatim so operators see "Failed to list polling targets"
      // (from the handler) or "Network error" (from the client).
      setError(err instanceof Error ? err.message : 'Failed to load polling targets');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const create = useCallback(
    async (input: PollingTargetInput): Promise<PollingTarget> => {
      const created = await api.post<PollingTarget>(ENDPOINT, input);
      // Re-fetch so the row order, audit columns, and any server-
      // generated defaults (id, collectorChain) reflect what's on
      // disk instead of a partial echo of the input.
      await refresh();
      return created;
    },
    [refresh],
  );

  const update = useCallback(
    async (id: string, input: PollingTargetInput): Promise<PollingTarget> => {
      const updated = await api.put<PollingTarget>(`${ENDPOINT}/${id}`, input);
      await refresh();
      return updated;
    },
    [refresh],
  );

  const remove = useCallback(
    async (id: string): Promise<void> => {
      await api.delete(`${ENDPOINT}/${id}`);
      await refresh();
    },
    [refresh],
  );

  return { targets, loading, error, refresh, create, update, remove };
}
