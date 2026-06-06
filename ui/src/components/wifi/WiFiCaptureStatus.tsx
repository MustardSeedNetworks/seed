import type { Status } from '../../types/generated/wifi-airspace-response';

interface WiFiCaptureStatusProps {
  status: Status;
}

interface Stat {
  label: string;
  value: number;
}

/**
 * WiFiCaptureStatus is a compact header showing whether monitor-mode capture is
 * active (and its source) plus the live entity counts. When capture is inactive
 * it states so plainly rather than implying an empty network.
 */
export function WiFiCaptureStatus({ status }: WiFiCaptureStatusProps) {
  const stats: Stat[] = [
    { label: 'SSIDs', value: status.ssids },
    { label: 'APs', value: status.aps },
    { label: 'BSSIDs', value: status.bsses },
    { label: 'Clients', value: status.stations },
  ];

  return (
    <div data-testid="wifi-capture-status" className="stack-sm">
      <div className="flex items-center gap-2">
        <span
          data-testid="wifi-capture-indicator"
          className={`inline-block h-2 w-2 rounded-full ${
            status.captureActive ? 'bg-status-success' : 'bg-text-muted'
          }`}
        />
        <span className="text-sm text-text-secondary">
          {status.captureActive ? (
            <>
              Live capture on <span className="font-medium text-text-primary">{status.source}</span>
            </>
          ) : (
            'Monitor capture inactive — showing the last observed airspace'
          )}
        </span>
      </div>
      <dl className="flex flex-wrap gap-4">
        {stats.map((s) => (
          <div key={s.label} className="flex items-baseline gap-1">
            <dd className="text-base font-semibold text-text-primary">{s.value}</dd>
            <dt className="text-xs text-text-muted">{s.label}</dt>
          </div>
        ))}
      </dl>
    </div>
  );
}
