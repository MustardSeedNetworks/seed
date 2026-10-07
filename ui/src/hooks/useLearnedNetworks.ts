/**
 * useLearnedNetworks
 *
 * The learned networks awaiting the operator's decision (seed#3108). Seed
 * learns target networks from routers' tables and its own routes but never
 * sweeps one until the operator adds it, so the list has to be put in front of
 * them rather than left as disabled rows in Settings.
 *
 * The endpoint is operator+ on read as well as write, so a viewer never asks.
 */

import {
  type UseMutationResult,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { api } from '../api';
import { useRole } from '../contexts/RoleContext';
import type { SubnetDecisionRequest } from '../types/generated/subnet-decision-request';
import type { SubnetResponse } from '../types/generated/subnet-response';

const PENDING_PATH = '/api/v1/security/devices/subnets/pending';

export const pendingNetworksKey = ['discovery', 'subnets', 'pending'] as const;

/** Pending learned networks, in the order they were learned. */
export function usePendingNetworks(): SubnetResponse[] {
  const { canWrite } = useRole();
  const { data } = useQuery({
    queryKey: pendingNetworksKey,
    queryFn: () => api.get<SubnetResponse[]>(PENDING_PATH),
    enabled: canWrite,
    // The learner runs once per rescan interval; a minute is soon enough to
    // offer what it found.
    refetchInterval: 60 * 1000,
  });
  return canWrite ? (data ?? []) : [];
}

/** Records Add or Dismiss for one learned network. */
export function useDecideNetwork(): UseMutationResult<void, Error, SubnetDecisionRequest> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: SubnetDecisionRequest): Promise<void> => {
      await api.post(PENDING_PATH, request);
    },
    // The decision is final for this list, so retrying a refusal adds nothing.
    retry: false,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: pendingNetworksKey }),
  });
}
