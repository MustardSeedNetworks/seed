/**
 * useCapabilities Hook
 *
 * Fetches and tracks system capability status from the backend.
 * Used to detect missing network capabilities (ICMP, packet capture, etc.)
 * and display appropriate warnings to users.
 *
 * Issue #803: UI detect/warn missing network capabilities
 */

import { useCallback, useEffect, useState } from 'react';
import { LogComponents, logger } from '../lib/logger';

const API_BASE = '';

export interface Capabilities {
  /** Whether raw ICMP sockets are available (requires root or CAP_NET_RAW) */
  icmpAvailable: boolean;
}

interface UseCapabilitiesResult {
  /** Current capability status */
  capabilities: Capabilities | null;
  /** Whether the initial fetch is in progress */
  loading: boolean;
  /** Any error that occurred during fetch */
  error: string | null;
  /** Re-fetch capabilities */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and track system capabilities.
 * Capabilities are fetched once on mount and can be refreshed manually.
 */
export function useCapabilities(): UseCapabilitiesResult {
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCapabilities = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/status`, {
        credentials: 'include',
      });

      if (!response.ok) {
        if (response.status === 401) {
          // Not authenticated yet, this is expected during login
          return;
        }
        throw new Error(`HTTP ${response.status}`);
      }

      const data: { icmpAvailable?: boolean } = await (response.json() as Promise<{
        icmpAvailable?: boolean;
      }>);

      setCapabilities({
        icmpAvailable: data.icmpAvailable === true,
      });
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch capabilities';
      logger.error(LogComponents.System, 'Failed to fetch capabilities', err);
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCapabilities().catch(() => undefined);
  }, [fetchCapabilities]);

  return {
    capabilities,
    loading,
    error,
    refresh: fetchCapabilities,
  };
}

/**
 * Returns a list of missing capabilities with descriptions.
 * Useful for displaying warnings to users.
 */
export function getMissingCapabilities(capabilities: Capabilities | null): Array<{
  id: string;
  title: string;
  description: string;
  remediation: string;
}> {
  if (!capabilities) {
    return [];
  }

  const missing: Array<{
    id: string;
    title: string;
    description: string;
    remediation: string;
  }> = [];

  if (!capabilities.icmpAvailable) {
    missing.push({
      id: 'icmp',
      title: 'ICMP Unavailable',
      description:
        'Raw ICMP sockets are not available. Gateway ping, traceroute, and other ICMP-based features will not work.',
      remediation:
        'Run The Seed with elevated privileges (sudo) or grant CAP_NET_RAW capability: sudo setcap cap_net_raw,cap_net_admin=+ep /path/to/seed',
    });
  }

  return missing;
}
