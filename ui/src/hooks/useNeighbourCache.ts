/**
 * useNeighbourCache — this device's own ARP/NDP neighbour cache (#328).
 *
 * Not to be confused with /api/v1/topology/arp, which serves SNMP-harvested
 * bindings from remote nodes. This is what the box in front of the operator can
 * see on its own link, which is what they need when an IP is not resolving to a
 * MAC on the segment they are plugged into.
 */

import { useCallback, useEffect, useState } from 'react';

import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type {
  NeighbourCacheResponse,
  NeighbourEntry,
} from '../types/generated/neighbour-cache-response';

interface NeighbourCache {
  entries: NeighbourEntry[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useNeighbourCache(): NeighbourCache {
  const [entries, setEntries] = useState<NeighbourEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.get<NeighbourCacheResponse>('/api/v1/network/neighbours');
      setEntries(response?.entries ?? []);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to read the neighbour cache';
      setError(message);
      logger.error(LogComponents.DISCOVERY, 'Failed to read the neighbour cache', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh().catch(() => undefined);
  }, [refresh]);

  return { entries, loading, error, refresh };
}
