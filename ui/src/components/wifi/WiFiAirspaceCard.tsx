import { useWifiAirspace } from '../../hooks/useWifiVisibility';
import { Card } from '../ui/card';
import { WiFiAirspaceTree } from './WiFiAirspaceTree';
import { WiFiCaptureStatus } from './WiFiCaptureStatus';

/**
 * WiFiAirspaceCard is the container for the live airspace tree: it polls the
 * Pro-gated /wifi/airspace endpoint and renders the capture status plus the
 * SSID -> AP -> BSSID -> client hierarchy.
 */
export function WiFiAirspaceCard() {
  const { data, isLoading, isError } = useWifiAirspace();

  const status = data?.status.captureActive ? 'success' : 'unknown';

  return (
    <Card
      title="Wi-Fi Airspace"
      subtitle="Live SSID / AP / BSSID / client map from 802.11 management-frame capture."
      status={status}
    >
      {isLoading ? (
        <p data-testid="wifi-airspace-loading" className="text-sm text-text-muted">
          Loading airspace…
        </p>
      ) : isError || !data ? (
        <p data-testid="wifi-airspace-error" className="text-sm text-text-muted">
          Airspace data is unavailable.
        </p>
      ) : (
        <div className="stack-md">
          <WiFiCaptureStatus status={data.status} />
          <WiFiAirspaceTree ssids={data.ssids} />
        </div>
      )}
    </Card>
  );
}
