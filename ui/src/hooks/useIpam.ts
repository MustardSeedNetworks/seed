/**
 * Data hook for the /api/v1/ipam surface (P-D5, UI-SEED-46).
 *
 * One read carries every network's and every DHCP scope's utilization; adding
 * or deleting a scope re-reads it, so the page shows what the server counts.
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { IPAMScopeRequest } from '../types/generated/ipam-scope-request';
import type { IPAMSubnetsResponse } from '../types/generated/ipam-subnets-response';

const SUBNETS = '/api/v1/ipam/subnets';
const SCOPES = '/api/v1/ipam/scopes';

export interface UseIpamResult {
  data: IPAMSubnetsResponse | null;
  error: string | null;
  addScope: (input: IPAMScopeRequest) => Promise<void>;
  deleteScope: (start: string) => Promise<void>;
}

export function useIpam(): UseIpamResult {
  const [data, setData] = useState<IPAMSubnetsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = async (): Promise<void> => {
    setError(null);
    await api
      .get<IPAMSubnetsResponse>(SUBNETS)
      .then(setData)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  };

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const addScope = async (input: IPAMScopeRequest): Promise<void> => {
    await api.post(SCOPES, input);
    await refresh();
  };

  const deleteScope = async (start: string): Promise<void> => {
    await api.delete(`${SCOPES}?start=${encodeURIComponent(start)}`);
    await refresh();
  };

  return { data, error, addScope, deleteScope };
}
