/**
 * useDriverStats — the NIC driver's own error counters (#416).
 *
 * Linux only. The endpoint refuses elsewhere with a 501 naming the reason, and
 * the card is wrapped in <FeatureUnavailable> so an operator on macOS sees that
 * explanation rather than an empty table.
 */

import { useCallback, useEffect, useState } from 'react';

import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { CuratedCounter, DriverStatsResponse } from '../types/generated/driver-stats-response';

interface DriverStats {
  counters: CuratedCounter[];
  total: number;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useDriverStats(interfaceName?: string): DriverStats {
  const [counters, setCounters] = useState<CuratedCounter[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const query = interfaceName ? `?interface=${encodeURIComponent(interfaceName)}` : '';
      const stats = await api.get<DriverStatsResponse>(
        `/api/v1/telemetry/interface/driver-stats${query}`,
      );
      setCounters(stats?.counters ?? []);
      setTotal(stats?.total ?? 0);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to read driver statistics';
      setError(message);
      logger.error(LogComponents.NETWORK, 'Failed to read driver statistics', err);
    } finally {
      setLoading(false);
    }
  }, [interfaceName]);

  useEffect(() => {
    refresh().catch(() => undefined);
  }, [refresh]);

  return { counters, total, loading, error, refresh };
}
