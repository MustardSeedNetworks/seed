import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { WiFiAirspaceResponse } from '../types/generated/wifi-airspace-response';
import type { WiFiAnomaliesResponse } from '../types/generated/wifi-anomalies-response';

// Live read endpoints for the Pro-gated Wi-Fi visibility feature. Both poll on a
// short interval so the airspace tree and anomaly stream track the capture loop.
const airspaceEndpoint = '/api/v1/wifi/airspace';
const anomaliesEndpoint = '/api/v1/wifi/anomalies';

// refetchIntervalMs balances freshness against load; the backend evaluates the
// airspace on a similar cadence.
const refetchIntervalMs = 10_000;

export const wifiVisibilityKeys = {
  all: ['wifi-visibility'] as const,
  airspace: () => [...wifiVisibilityKeys.all, 'airspace'] as const,
  anomalies: () => [...wifiVisibilityKeys.all, 'anomalies'] as const,
};

/** useWifiAirspace polls the cross-referenced SSID -> AP -> BSSID -> client tree. */
export function useWifiAirspace() {
  return useQuery({
    queryKey: wifiVisibilityKeys.airspace(),
    queryFn: () => api.get<WiFiAirspaceResponse>(airspaceEndpoint),
    refetchInterval: refetchIntervalMs,
  });
}

/** useWifiAnomalies polls the deduped, severity-ranked Wi-Fi anomaly stream. */
export function useWifiAnomalies() {
  return useQuery({
    queryKey: wifiVisibilityKeys.anomalies(),
    queryFn: () => api.get<WiFiAnomaliesResponse>(anomaliesEndpoint),
    refetchInterval: refetchIntervalMs,
  });
}
