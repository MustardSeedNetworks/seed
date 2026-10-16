/**
 * Data hooks for the /api/v1/device-groups surface (P-D10, UI-SEED-25).
 *
 * useDeviceGroups is the list and its three mutations, refreshed after each so
 * the page shows what the server stored. useDeviceGroupSummary is one group's
 * dashboard read: each member's last poll and the members' open alerts.
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { DeviceGroupListResponse } from '../types/generated/device-group-list-response';
import type { DeviceGroupRequest } from '../types/generated/device-group-request';
import type { DeviceGroupResponse } from '../types/generated/device-group-response';
import type { DeviceGroupSummaryResponse } from '../types/generated/device-group-summary-response';

const ENDPOINT = '/api/v1/device-groups';

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export interface UseDeviceGroupsResult {
  groups: DeviceGroupResponse[];
  loading: boolean;
  error: string | null;
  create: (input: DeviceGroupRequest) => Promise<DeviceGroupResponse>;
  update: (id: string, input: DeviceGroupRequest) => Promise<DeviceGroupResponse>;
  remove: (id: string) => Promise<void>;
}

export function useDeviceGroups(): UseDeviceGroupsResult {
  const [groups, setGroups] = useState<DeviceGroupResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = async (): Promise<void> => {
    setError(null);
    await api
      .get<DeviceGroupListResponse>(ENDPOINT)
      .then((resp) => setGroups(resp.groups))
      .catch((err: unknown) => setError(message(err)));
    setLoading(false);
  };

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const create = async (input: DeviceGroupRequest): Promise<DeviceGroupResponse> => {
    const created = await api.post<DeviceGroupResponse>(ENDPOINT, input);
    await refresh();
    return created;
  };

  const update = async (id: string, input: DeviceGroupRequest): Promise<DeviceGroupResponse> => {
    const updated = await api.put<DeviceGroupResponse>(
      `${ENDPOINT}/${encodeURIComponent(id)}`,
      input,
    );
    await refresh();
    return updated;
  };

  const remove = async (id: string): Promise<void> => {
    await api.delete(`${ENDPOINT}/${encodeURIComponent(id)}`);
    await refresh();
  };

  return { groups, loading, error, create, update, remove };
}

export interface UseDeviceGroupSummaryResult {
  summary: DeviceGroupSummaryResponse | null;
  error: string | null;
}

/**
 * Reads the summary of group `id`, again whenever `revision` changes, so an
 * edit to the group's members shows in its summary without a reload.
 */
export function useDeviceGroupSummary(
  id: string | null,
  revision: string,
): UseDeviceGroupSummaryResult {
  const [summary, setSummary] = useState<DeviceGroupSummaryResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setSummary(null);
    setError(null);
    if (!id) {
      return;
    }
    let cancelled = false;
    api
      .get<DeviceGroupSummaryResponse>(`${ENDPOINT}/${encodeURIComponent(id)}/summary`)
      .then((resp) => {
        if (!cancelled) {
          setSummary(resp);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(message(err));
        }
      });
    return (): void => {
      cancelled = true;
    };
  }, [id, revision]);

  return { summary, error };
}
