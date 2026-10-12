/**
 * Channel Graph Hook
 *
 * Manages WiFi channel graph data fetching and state
 * for the WiFi channel visualization component.
 *
 * Extracted from App.tsx to reduce component complexity (#889).
 */

import { useState } from 'react';
import { api } from '../api';
import type { WiFiChannelGraphResponse } from '../types/generated/wifi-channel-graph-response';

interface UseChannelGraphProps {
  /** Whether the current interface is WiFi */
  isWifi: boolean;
  /** Current interface name */
  currentInterface: string;
}

interface UseChannelGraphReturn {
  /** Channel graph data (null if not loaded) */
  channelGraphData: WiFiChannelGraphResponse | null;
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
  const [channelGraphData, setChannelGraphData] = useState<WiFiChannelGraphResponse | null>(null);
  const [channelGraphLoading, setChannelGraphLoading] = useState(false);

  // Fetch channel graph data for WiFi visualization
  const fetchChannelGraphData = async () => {
    if (!(isWifi && currentInterface)) {
      return;
    }
    setChannelGraphLoading(true);
    try {
      setChannelGraphData(
        await api.get<WiFiChannelGraphResponse>(
          `/api/v1/wifi/wifi/channel-graph?interface=${currentInterface}`,
        ),
      );
    } catch {
      setChannelGraphData({
        interface: currentInterface,
        available: false,
        error: 'Failed to fetch channel data',
      });
    }
    setChannelGraphLoading(false);
  };

  return {
    channelGraphData,
    channelGraphLoading,
    fetchChannelGraphData,
  };
}

export default useChannelGraph;
