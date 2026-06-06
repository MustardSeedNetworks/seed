import { useWifiAnomalies } from '../../hooks/useWifiVisibility';
import type { Anomaly } from '../../types/generated/wifi-anomalies-response';
import { Card } from '../ui/card';
import type { Status } from '../ui/statusConfig';
import { WiFiAnomalyStream } from './WiFiAnomalyStream';

// cardStatus reflects the most urgent anomaly severity in the card header.
function cardStatus(anomalies: Anomaly[]): Status {
  if (anomalies.some((a) => a.severity === 'critical')) {
    return 'error';
  }
  if (anomalies.some((a) => a.severity === 'warning')) {
    return 'warning';
  }
  return 'success';
}

/**
 * WiFiAnomaliesCard is the container for the Wi-Fi anomaly stream: it polls the
 * Pro-gated /wifi/anomalies endpoint and renders the severity-ranked detections.
 */
export function WiFiAnomaliesCard() {
  const { data, isLoading, isError } = useWifiAnomalies();

  return (
    <Card
      title="Wi-Fi Anomalies"
      subtitle="Security, RF, roaming, and standards anomalies detected in the airspace."
      status={data ? cardStatus(data.anomalies) : 'unknown'}
    >
      {isLoading ? (
        <p data-testid="wifi-anomalies-loading" className="text-sm text-text-muted">
          Loading anomalies…
        </p>
      ) : isError || !data ? (
        <p data-testid="wifi-anomalies-error" className="text-sm text-text-muted">
          Anomaly data is unavailable.
        </p>
      ) : (
        <WiFiAnomalyStream anomalies={data.anomalies} />
      )}
    </Card>
  );
}
