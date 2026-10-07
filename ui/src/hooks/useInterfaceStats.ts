/**
 * Data hook for GET /api/v1/topology/interfaces: every interface the polling
 * targets have reported, with its latest rates (UI-SEED-21).
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type {
  InterfaceStatsListResponse,
  InterfaceStatsResponse,
} from '../types/generated/interface-stats-list-response';

const ENDPOINT = '/api/v1/topology/interfaces';

export interface UseInterfaceStatsResult {
  interfaces: InterfaceStatsResponse[];
  loading: boolean;
  error: string | null;
}

export function useInterfaceStats(): UseInterfaceStatsResult {
  const [interfaces, setInterfaces] = useState<InterfaceStatsResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .get<InterfaceStatsListResponse>(ENDPOINT)
      .then((resp) => {
        if (!cancelled) {
          setInterfaces(resp.interfaces);
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
  }, []);

  return { interfaces, loading, error };
}
