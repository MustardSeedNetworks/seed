/**
 * Data hook for the flow explorer (UI-SEED-23): the top talkers, conversations
 * and applications over one window, ranked one way. The three reads resolve
 * the same window on the server, so the first response's window stands for
 * all three.
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { FlowApplicationsResponse } from '../types/generated/flow-applications-response';
import type { FlowConversationsResponse } from '../types/generated/flow-conversations-response';
import type { FlowTalkersResponse } from '../types/generated/flow-talkers-response';

export const FLOW_RANGES = ['1d', '7d', '30d', '90d'] as const;
export type FlowRange = (typeof FLOW_RANGES)[number];

export const FLOW_RANKS = ['bytes', 'packets'] as const;
export type FlowRank = (typeof FLOW_RANKS)[number];

export interface FlowTops {
  talkers: FlowTalkersResponse;
  conversations: FlowConversationsResponse;
  applications: FlowApplicationsResponse;
}

export interface UseFlowTopsResult {
  tops: FlowTops | null;
  loading: boolean;
  error: string | null;
}

export function useFlowTops(range: FlowRange, by: FlowRank): UseFlowTopsResult {
  const [tops, setTops] = useState<FlowTops | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const query = new URLSearchParams({ range, by }).toString();
    setLoading(true);
    setError(null);
    Promise.all([
      api.get<FlowTalkersResponse>(`/api/v1/flows/top-talkers?${query}`),
      api.get<FlowConversationsResponse>(`/api/v1/flows/top-conversations?${query}`),
      api.get<FlowApplicationsResponse>(`/api/v1/flows/top-applications?${query}`),
    ])
      .then(([talkers, conversations, applications]) => {
        if (!cancelled) {
          setTops({ talkers, conversations, applications });
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
  }, [range, by]);

  return { tops, loading, error };
}
