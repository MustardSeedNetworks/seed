import { useTranslation } from 'react-i18next';
import { useSettings } from '../../contexts/useSettings';
import { cn, icon as iconTokens, layout, spacing } from '../../styles/theme';
import { CardDivider, CardRow, CardValue, type Status } from '../ui/card';
import { Wifi } from '../ui/icons';
import { SimpleBaseCard } from './BaseCard';

/**
 * Current WiFi connection information
 */
interface WiFiAssociation {
  status: 'associated';
  ssid: string; // Network name (Service Set Identifier)
  bssid: string; // Access point MAC address
  signal: number; // Signal strength in dBm (negative value)
  channel: number; // WiFi channel (1-13 for 2.4GHz, 36+ for 5GHz)
  frequency: number; // Frequency in MHz (2400-2500 or 5000-6000)
  security: string; // Security protocol (WPA2, WPA3, Open, etc.)
}

export type WiFiData =
  | WiFiAssociation
  | { status: 'detailsWithheld'; reason: string; remediation: string }
  | { status: 'notAssociated' };

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
  thresholds: { warning: number; critical: number },
): Status {
  if (signal <= thresholds.critical) {
    return 'error';
  }
  if (signal <= thresholds.warning) {
    return 'warning';
  }
  return 'success';
}

function signalToPercentage(signal: number): number {
  // Rough conversion: -30 dBm = 100%, -90 dBm = 0%
  const percent = Math.min(100, Math.max(0, ((signal + 90) / 60) * 100));
  return Math.round(percent);
}

function getSignalBars(signal: number): string {
  const percent = signalToPercentage(signal);
  if (percent >= 75) {
    return '▂▄▆█';
  }
  if (percent >= 50) {
    return '▂▄▆░';
  }
  if (percent >= 25) {
    return '▂▄░░';
  }
  return '▂░░░';
}

/**
 * Displays current WiFi connection status with signal strength visualization.
 */
export function WiFiCard({
  data,
  loading,
  visible = true,
}: WiFiCardProps): React.JSX.Element | null {
  const { t: tr } = useTranslation('cards');
  const { t: tc } = useTranslation('common');
  const { thresholds } = useSettings();
  // Map context ThresholdPair (good/warning) to card format (warning/critical)
  // For WiFi: good = -50 dBm, warning = -70 dBm (higher is better, so critical = warning)
  // Use defaults if thresholds not yet loaded
  const th = {
    warning: thresholds?.wifi?.good ?? -50,
    critical: thresholds?.wifi?.warning ?? -70,
  };

  // Don't render if not on WiFi
  if (!visible) {
    return null;
  }

  const status = data?.status === 'associated' ? getSignalStatus(data.signal, th) : 'unknown';

  return (
    <SimpleBaseCard
      title={tr('wifi.title')}
      icon={<Wifi className={iconTokens.size.md} />}
      status={loading ? 'loading' : status}
      loading={loading}
      loadingContent={<CardValue value={tc('status.scanning')} size="lg" />}
    >
      {data?.status === 'associated' ? (
        <div data-testid="wifi-associated">
          <CardValue value={data.ssid} size="lg" />
          <div className={cn(layout.inline.default, spacing.margin.top.tight)}>
            <span className="body-large font-mono">{getSignalBars(data.signal)}</span>
            <span className="body-small text-text-muted">
              {data.signal} dBm ({signalToPercentage(data.signal)}%)
            </span>
          </div>
          <CardDivider />
          <CardRow label={tr('wifi.bssid')} value={data.bssid} />
          <CardRow label={tr('wifi.channel')} value={data.channel.toString()} />
          <CardRow label={tr('wifi.frequency')} value={`${data.frequency} MHz`} />
          <CardRow label={tr('wifi.security')} value={data.security} />
        </div>
      ) : data?.status === 'detailsWithheld' ? (
        <div data-testid="wifi-details-withheld" className={spacing.stack.sm}>
          <CardValue value={tr('wifi.detailsWithheld.title')} size="md" />
          <p className="body-small text-text-muted">{tr('wifi.detailsWithheld.reason')}</p>
          <p className="body-small text-text-muted">{tr('wifi.detailsWithheld.remediation')}</p>
        </div>
      ) : (
        <div data-testid="wifi-not-associated">
          <CardValue value={tc(data ? 'status.disconnected' : 'status.unavailable')} size="md" />
        </div>
      )}
    </SimpleBaseCard>
  );
}
