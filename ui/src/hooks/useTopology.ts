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
  TopologyLink,
  TopologyLinksResponse,
  TopologyNode,
  TopologyNodeDetailResponse,
  TopologyNodesResponse,
} from '../types/topology';

const ENDPOINT = '/api/v1/topology';

/** The handler defaults to 200 rows and caps at 1000 (topologyDefaultLimit /
 * topologyMaxLimit). The map has to show the whole discovered network, not
 * its first page, so both reads ask for the cap. */
const PAGE_LIMIT = 1000;

export interface UseTopologyNodesResult {
  nodes: TopologyNode[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/** useTopologyNodes lists every node visible to the current session.
 * The endpoint supports filtering (device_type, since); neither is
 * exposed in this hook yet — the page just renders everything. Filters
 * land when the operator UX needs them. */
export function useTopologyNodes(): UseTopologyNodesResult {
  const [nodes, setNodes] = useState<TopologyNode[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.get<TopologyNodesResponse>(`${ENDPOINT}/nodes?limit=${PAGE_LIMIT}`);
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

export interface UseTopologyLinksResult {
  links: TopologyLink[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/** useTopologyLinks lists every edge the reconcilers hold. Separate from the
 * node list because the map needs both and the list pane needs only nodes —
 * one hook returning both would make the list pay for the edges. */
export function useTopologyLinks(): UseTopologyLinksResult {
  const [links, setLinks] = useState<TopologyLink[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.get<TopologyLinksResponse>(`${ENDPOINT}/links?limit=${PAGE_LIMIT}`);
      setLinks(resp.links ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load links');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { links, loading, error, refresh };
}
