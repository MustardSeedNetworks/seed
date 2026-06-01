/**
 * Data hook for the /api/v1/alert-rules surface (#1384).
 *
 * Same framework-light pattern as usePollingTargets — fetch on
 * mount, expose CRUD mutations that re-fetch after writing so the
 * UI shows server-canonical state.
 */

import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { AlertRule, AlertRuleInput, AlertRulesListResponse } from '../types/alertRules';

const ENDPOINT = '/api/v1/alert-rules';

export interface UseAlertRulesResult {
  rules: AlertRule[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  create: (input: AlertRuleInput) => Promise<AlertRule>;
  update: (id: number, input: AlertRuleInput) => Promise<AlertRule>;
  remove: (id: number) => Promise<void>;
}

export function useAlertRules(): UseAlertRulesResult {
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.get<AlertRulesListResponse>(ENDPOINT);
      setRules(resp.rules ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load alert rules');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const create = useCallback(
    async (input: AlertRuleInput): Promise<AlertRule> => {
      const created = await api.post<AlertRule>(ENDPOINT, input);
      await refresh();
      return created;
    },
    [refresh],
  );

  const update = useCallback(
    async (id: number, input: AlertRuleInput): Promise<AlertRule> => {
      const updated = await api.put<AlertRule>(`${ENDPOINT}/${id}`, input);
      await refresh();
      return updated;
    },
    [refresh],
  );

  const remove = useCallback(
    async (id: number): Promise<void> => {
      await api.delete(`${ENDPOINT}/${id}`);
      await refresh();
    },
    [refresh],
  );

  return { rules, loading, error, refresh, create, update, remove };
}
