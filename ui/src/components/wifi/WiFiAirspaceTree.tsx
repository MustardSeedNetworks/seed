import type { APGroup, BSSView, SSIDGroup } from '../../types/generated/wifi-airspace-response';

interface WiFiAirspaceTreeProps {
  ssids: SSIDGroup[];
}

function ssidLabel(g: SSIDGroup): string {
  if (g.ssid !== '') {
    return g.ssid;
  }
  return g.hidden ? '(hidden SSID)' : '(unnamed)';
}

function StationRows({ bss }: { bss: BSSView }) {
  if (bss.stations.length === 0) {
    return null;
  }
  return (
    <ul className="ml-4 stack-2xs">
      {bss.stations.map((s) => (
        <li key={s.mac} data-testid="wifi-station" className="text-xs text-text-muted">
          <span className="font-mono">{s.mac}</span> · {s.signalDbm} dBm · {s.frames} frames
        </li>
      ))}
    </ul>
  );
}

function BSSRows({ ap }: { ap: APGroup }) {
  return (
    <ul className="ml-4 stack-2xs">
      {ap.bsses.map((b) => (
        <li key={b.bssid} data-testid="wifi-bss" className="stack-2xs">
          <p className="text-xs text-text-secondary">
            <span className="font-mono">{b.bssid}</span> · {b.band} ch {b.channel}
            {b.channelWidthMhz > 0 ? ` (${b.channelWidthMhz} MHz)` : ''} · {b.security} ·{' '}
            {b.standard} · {b.signalDbm} dBm
            {b.hasBssLoad ? ` · ${Math.round((b.channelUtil / 255) * 100)}% util` : ''}
          </p>
          <StationRows bss={b} />
        </li>
      ))}
    </ul>
  );
}

/**
 * WiFiAirspaceTree renders the cross-referenced SSID -> AP -> BSSID -> client
 * hierarchy as collapsible sections. Protocol terms (SSID, BSSID, dBm, 802.11
 * standards) are shown verbatim.
 */
export function WiFiAirspaceTree({ ssids }: WiFiAirspaceTreeProps) {
  if (ssids.length === 0) {
    return (
      <p data-testid="wifi-airspace-empty" className="text-sm text-text-muted">
        No Wi-Fi networks observed yet.
      </p>
    );
  }

  return (
    <div data-testid="wifi-airspace-tree" className="stack-xs">
      {ssids.map((g) => (
        <details
          key={ssidLabel(g) + g.bssCount}
          data-testid="wifi-ssid-group"
          className="rounded-md border border-surface-border bg-surface-base p-2"
        >
          <summary className="cursor-pointer text-sm font-medium text-text-primary">
            {ssidLabel(g)}{' '}
            <span className="text-xs font-normal text-text-muted">
              ({g.apCount} AP / {g.bssCount} BSSID / {g.stationCount} clients)
            </span>
          </summary>
          <div className="mt-2 stack-xs">
            {g.aps.map((ap) => (
              <div key={ap.key} data-testid="wifi-ap-group" className="stack-2xs">
                <p className="text-xs font-medium text-text-secondary">
                  AP {ap.vendor ? `· ${ap.vendor}` : ''}
                </p>
                <BSSRows ap={ap} />
              </div>
            ))}
          </div>
        </details>
      ))}
    </div>
  );
}
