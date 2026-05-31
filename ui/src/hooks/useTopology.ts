/**
 * Hooks for /api/v1/topology/* — list view + node detail view.
 *
 * Two hooks rather than one so the list page doesn't accidentally
 * fetch the much heavier detail payload, and so the detail page
 * stays mounted across list refreshes when the user clicks back.
 */

import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type {
  TopologyNode,
  TopologyNodeDetailResponse,
  TopologyNodesResponse,
} from '../types/topology';

const ENDPOINT = '/api/v1/topology';

export interface UseTopologyNodesResult {
  nodes: TopologyNode[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/** useTopologyNodes lists every node visible to the current session.
 * The endpoint supports filtering (device_type, since, limit); none
 * of those are exposed in this hook yet — the page just renders
 * everything. Filters land when the operator UX needs them. */
export function useTopologyNodes(): UseTopologyNodesResult {
  const [nodes, setNodes] = useState<TopologyNode[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.get<TopologyNodesResponse>(`${ENDPOINT}/nodes`);
      setNodes(resp.nodes ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load topology');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { nodes, loading, error, refresh };
}

export interface UseTopologyNodeResult {
  detail: TopologyNodeDetailResponse | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/** useTopologyNode loads one node plus its interfaces and links in
 * one HTTP call (the handler returns the bundled payload). Pass an
 * empty id to render an empty-state without firing a request. */
export function useTopologyNode(id: string): UseTopologyNodeResult {
  const [detail, setDetail] = useState<TopologyNodeDetailResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    if (!id) {
      setDetail(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const resp = await api.get<TopologyNodeDetailResponse>(`${ENDPOINT}/nodes/${id}`);
      setDetail(resp);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load node');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { detail, loading, error, refresh };
}
