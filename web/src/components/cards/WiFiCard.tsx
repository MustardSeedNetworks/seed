import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";

export interface WiFiData {
  ssid: string;
  bssid: string;
  signal: number; // dBm
  channel: number;
  frequency: number; // MHz
  security: string;
}

interface WiFiCardProps {
  data: WiFiData | null;
  loading?: boolean;
  thresholds?: { warning: number; critical: number };
  visible?: boolean;
}

function getSignalStatus(
  signal: number,
  thresholds: { warning: number; critical: number },
): Status {
  if (signal <= thresholds.critical) return "error";
  if (signal <= thresholds.warning) return "warning";
  return "success";
}

function signalToPercentage(signal: number): number {
  // Rough conversion: -30 dBm = 100%, -90 dBm = 0%
  const percent = Math.min(100, Math.max(0, ((signal + 90) / 60) * 100));
  return Math.round(percent);
}

function getSignalBars(signal: number): string {
  const percent = signalToPercentage(signal);
  if (percent >= 75) return "▂▄▆█";
  if (percent >= 50) return "▂▄▆░";
  if (percent >= 25) return "▂▄░░";
  return "▂░░░";
}

export function WiFiCard({
  data,
  loading,
  thresholds,
  visible = true,
}: WiFiCardProps) {
  const t = thresholds || { warning: -70, critical: -80 };

  // Don't render if not on WiFi
  if (!visible) {
    return null;
  }

  if (loading) {
    return (
      <Card title="Wi-Fi" status="loading">
        <CardValue value="Scanning..." size="lg" />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card title="Wi-Fi" status="unknown">
        <CardValue value="Not connected" size="md" />
      </Card>
    );
  }

  const status = getSignalStatus(data.signal, t);

  return (
    <Card title="Wi-Fi" status={status}>
      <CardValue value={data.ssid} size="lg" />
      <div className="flex items-center gap-2 mt-1">
        <span className="text-lg font-mono">{getSignalBars(data.signal)}</span>
        <span className="text-sm text-text-muted">
          {data.signal} dBm ({signalToPercentage(data.signal)}%)
        </span>
      </div>
      <CardDivider />
      <CardRow label="BSSID" value={data.bssid} />
      <CardRow label="Channel" value={data.channel.toString()} />
      <CardRow label="Frequency" value={`${data.frequency} MHz`} />
      <CardRow label="Security" value={data.security} />
    </Card>
  );
}
