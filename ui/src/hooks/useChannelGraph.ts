/**
 * Channel Graph Hook
 *
 * Manages WiFi channel graph data fetching and state
 * for the WiFi channel visualization component.
 *
 * Extracted from App.tsx to reduce component complexity (#889).
 */

import { useCallback, useState } from 'react';
import { api } from '../api';

/** Network data for channel graph visualization */
export type ChannelGraphNetwork = {
  ssid: string;
  bssid: string;
  channel: number;
  centerFreq: number;
  channelWidth: number;
  signal: number;
  band: string;
  isConnected: boolean;
};

/** Parsed channel graph data with networks grouped by band */
export type ChannelGraphData = {
  networks24Ghz: ChannelGraphNetwork[];
  networks5Ghz: ChannelGraphNetwork[];
  networks6Ghz: ChannelGraphNetwork[];
  connectedBssid?: string;
  scanTime: string;
};

/** Normalized response for channel graph component */
export type ChannelGraphResponse = {
  available: boolean;
  error?: string;
  data?: ChannelGraphData;
};

/** Raw API response for channel graph data */
type ChannelGraphApiResponse = {
  available: boolean;
  error?: string;
  data?: Record<string, unknown>;
};

/**
 * Normalizes the API response to ensure consistent structure.
 * Converts snake_case API fields to camelCase.
 */
const normalizeChannelGraphResponse = (response: ChannelGraphApiResponse): ChannelGraphResponse => {
  if (!response.data) {
    return response;
  }

  const data = response.data as Record<string, unknown>;
  const asNetworkArray = (value: unknown): ChannelGraphNetwork[] =>
    Array.isArray(value) ? (value as ChannelGraphNetwork[]) : [];

  return {
    ...response,
    data: {
      networks24Ghz: asNetworkArray(data.networks_2_4ghz),
      networks5Ghz: asNetworkArray(data.networks_5ghz),
      networks6Ghz: asNetworkArray(data.networks_6ghz),
      connectedBssid: typeof data.connected_bssid === 'string' ? data.connected_bssid : undefined,
      scanTime: typeof data.scan_time === 'string' ? data.scan_time : '',
    },
  };
};

interface UseChannelGraphProps {
  /** Whether the current interface is WiFi */
  isWifi: boolean;
  /** Current interface name */
  currentInterface: string;
}

interface UseChannelGraphReturn {
  /** Channel graph data (null if not loaded) */
  channelGraphData: ChannelGraphResponse | null;
  /** Whether channel graph data is loading */
  channelGraphLoading: boolean;
  /** Fetch channel graph data */
  fetchChannelGraphData: () => Promise<void>;
}

/**
 * Hook for managing WiFi channel graph data.
 *
 * @param props - Configuration with isWifi and currentInterface
 * @returns Object with channel graph state and fetch function
 */
export function useChannelGraph({
  isWifi,
  currentInterface,
}: UseChannelGraphProps): UseChannelGraphReturn {
  const [channelGraphData, setChannelGraphData] = useState<ChannelGraphResponse | null>(null);
  const [channelGraphLoading, setChannelGraphLoading] = useState(false);

  // Fetch channel graph data for WiFi visualization
  const fetchChannelGraphData = useCallback(async () => {
    if (!(isWifi && currentInterface)) {
      return;
    }
    setChannelGraphLoading(true);
    try {
      const response = await api.get<ChannelGraphApiResponse>(
        `/api/v1/canopy/wifi/channel-graph?interface=${currentInterface}`,
      );
      setChannelGraphData(normalizeChannelGraphResponse(response));
    } catch {
      setChannelGraphData({ available: false, error: 'Failed to fetch channel data' });
    } finally {
      setChannelGraphLoading(false);
    }
  }, [isWifi, currentInterface]);

  return {
    channelGraphData,
    channelGraphLoading,
    fetchChannelGraphData,
  };
}

export default useChannelGraph;
