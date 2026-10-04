import { useTranslation } from 'react-i18next';
import { useWifiAnomalies } from '../../hooks/useWifiVisibility';
import type { WiFiAnomaliesResponse } from '../../types/generated/wifi-anomalies-response';
import { Card } from '../ui/Card';
import type { Status } from '../ui/statusConfig';
import { WiFiAnomalyStream } from './WiFiAnomalyStream';
import { WiFiNeedsCapture } from './WiFiNeedsCapture';

// cardStatus reflects the most urgent anomaly severity in the card header. The
// Status vocabulary tops out at 'error', so both critical and error anomalies
// map to it (the per-row badge in severity.ts keeps them visually distinct).
// With nothing detected, rules that need a capture source did not run, so the
// result is not a clean bill of health.
function cardStatus({ anomalies, status }: WiFiAnomaliesResponse): Status {
  if (anomalies.some((a) => a.severity === 'critical' || a.severity === 'error')) {
    return 'error';
  }
  if (anomalies.some((a) => a.severity === 'warning')) {
    return 'warning';
  }
  return status.needsCapture?.length ? 'unknown' : 'success';
}

/**
 * WiFiAnomaliesCard is the container for the Wi-Fi anomaly stream: it polls the
 * Pro-gated /wifi/anomalies endpoint and renders the severity-ranked detections.
 */
export function WiFiAnomaliesCard() {
  const { t } = useTranslation('pages');
  const { data, isLoading, isError } = useWifiAnomalies();

  return (
    <Card
      title={t('wifi.anomaliesTitle')}
      subtitle={t('wifi.anomaliesSubtitle')}
      status={data ? cardStatus(data) : 'unknown'}
    >
      {isLoading ? (
        <p data-testid="wifi-anomalies-loading" className="text-sm text-text-muted">
          {t('wifi.anomaliesLoading')}
        </p>
      ) : isError || !data ? (
        <p data-testid="wifi-anomalies-error" className="text-sm text-text-muted">
          {t('wifi.anomaliesUnavailable')}
        </p>
      ) : (
        <div className="stack-md">
          <WiFiAnomalyStream anomalies={data.anomalies} />
          <WiFiNeedsCapture rules={data.status.needsCapture} />
        </div>
      )}
    </Card>
  );
}
