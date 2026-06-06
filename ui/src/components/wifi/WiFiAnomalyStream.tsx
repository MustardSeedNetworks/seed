import type { Anomaly } from '../../types/generated/wifi-anomalies-response';
import { severityStyle } from './severity';

interface WiFiAnomalyStreamProps {
  anomalies: Anomaly[];
}

/**
 * WiFiAnomalyStream renders the deduped Wi-Fi anomaly stream, most urgent first.
 * Each entry shows its severity, title, the subject it concerns, the
 * recommendation, and how many times it has recurred. Protocol/standard terms
 * (SSID, BSSID, 802.11, WPA3) are shown verbatim.
 */
export function WiFiAnomalyStream({ anomalies }: WiFiAnomalyStreamProps) {
  if (anomalies.length === 0) {
    return (
      <p data-testid="wifi-anomalies-empty" className="text-sm text-text-muted">
        No Wi-Fi anomalies detected.
      </p>
    );
  }

  const sorted = [...anomalies].sort(
    (a, b) => severityStyle(b.severity).rank - severityStyle(a.severity).rank,
  );

  return (
    <ul data-testid="wifi-anomaly-stream" className="stack-sm">
      {sorted.map((a) => (
        <li
          key={`${a.defKey}:${a.subject.kind}:${a.subject.id}`}
          data-testid="wifi-anomaly-row"
          className="rounded-md border border-surface-border bg-surface-base p-3 stack-2xs"
        >
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <span
                data-testid="wifi-anomaly-severity"
                className={`rounded-full px-2 py-0.5 text-xs font-medium uppercase ${severityStyle(a.severity).badge}`}
              >
                {a.severity}
              </span>
              <span className="text-sm font-medium text-text-primary">{a.title}</span>
            </div>
            {a.count > 1 ? <span className="text-xs text-text-muted">x{a.count}</span> : null}
          </div>
          <p className="text-xs text-text-secondary">
            {a.subject.kind}: <span className="font-mono">{a.subject.id}</span>
          </p>
          <p className="text-xs text-text-muted">{a.recommendation}</p>
        </li>
      ))}
    </ul>
  );
}
