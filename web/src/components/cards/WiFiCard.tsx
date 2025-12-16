/**
 * WiFi Connection Status Card Component
 *
 * Displays current WiFi connection information and signal strength.
 *
 * Features:
 * - SSID (network name) display
 * - BSSID (access point MAC) identification
 * - Signal strength in dBm with visual representation
 * - Signal bars (▂▄▆█) visual indicator
 * - WiFi channel and frequency information
 * - Security protocol display
 * - Threshold-based status coloring
 * - Only visible when connected to WiFi
 *
 * Signal Strength Conversion:
 * - Typical range: -30 dBm (excellent, very close) to -90 dBm (poor, far away)
 * - Displayed as percentage (0-100%) for easy interpretation
 * - Visual bars updated in real-time
 *
 * Status Indicators:
 * - **Success (Green)**: Signal -50 dBm or better (strong signal)
 * - **Warning (Yellow)**: Signal -50 to -70 dBm (acceptable but degrading)
 * - **Error (Red)**: Signal -70 dBm or worse (poor signal)
 *
 * The card is conditionally hidden when not connected to WiFi.
 */

import { useTranslation } from "react-i18next";
import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { useSettings } from "../../contexts/useSettings";
import { SimpleBaseCard } from "./BaseCard";
import { Wifi } from "../ui/Icons";
import { layout, icon as iconTokens } from "../../styles/theme";

/**
 * Current WiFi connection information
 */
export interface WiFiData {
  ssid: string; // Network name (Service Set Identifier)
  bssid: string; // Access point MAC address
  signal: number; // Signal strength in dBm (negative value)
  channel: number; // WiFi channel (1-13 for 2.4GHz, 36+ for 5GHz)
  frequency: number; // Frequency in MHz (2400-2500 or 5000-6000)
  security: string; // Security protocol (WPA2, WPA3, Open, etc.)
}

/**
 * Props for WiFi Card
 */
interface WiFiCardProps {
  data: WiFiData | null; // Current WiFi connection
  loading?: boolean; // True while loading data
  visible?: boolean; // If false, card is not rendered (not on WiFi)
}

/**
 * Determines card status based on signal strength and thresholds.
 * Lower dBm (more negative) = weaker signal.
 *
 * @param signal - Signal strength in dBm (negative value)
 * @param thresholds - Good and warning dBm thresholds
 * @returns Status indicator ('success', 'warning', 'error')
 */
function getSignalStatus(
  signal: number,
  thresholds: { warning: number; critical: number }
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

/**
 * Displays current WiFi connection status with signal strength visualization.
 */
export function WiFiCard({ data, loading, visible = true }: WiFiCardProps) {
  const { t: tr } = useTranslation("cards");
  const { t: tc } = useTranslation("common");
  const { thresholds } = useSettings();
  // Map context ThresholdPair (good/warning) to card format (warning/critical)
  // For WiFi: good = -50 dBm, warning = -70 dBm (higher is better, so critical = warning)
  const th = {
    warning: thresholds.wifi.good,
    critical: thresholds.wifi.warning,
  };

  // Don't render if not on WiFi
  if (!visible) {
    return null;
  }

  const status = data ? getSignalStatus(data.signal, th) : "unknown";

  return (
    <SimpleBaseCard
      title={tr("wifi.title")}
      icon={<Wifi className={iconTokens.size.md} />}
      status={loading ? "loading" : status}
      loading={loading}
      loadingContent={<CardValue value={tc("status.scanning")} size="lg" />}
    >
      {!data ? (
        <CardValue value={tc("status.disconnected")} size="md" />
      ) : (
        <>
          <CardValue value={data.ssid} size="lg" />
          <div className={`${layout.inline.default} mt-1`}>
            <span className="body-large font-mono">{getSignalBars(data.signal)}</span>
            <span className="body-small text-text-muted">
              {data.signal} dBm ({signalToPercentage(data.signal)}%)
            </span>
          </div>
          <CardDivider />
          <CardRow label={tr("wifi.bssid")} value={data.bssid} />
          <CardRow label={tr("wifi.channel")} value={data.channel.toString()} />
          <CardRow label={tr("wifi.frequency")} value={`${data.frequency} MHz`} />
          <CardRow label={tr("wifi.security")} value={data.security} />
        </>
      )}
    </SimpleBaseCard>
  );
}
