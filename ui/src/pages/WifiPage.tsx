import { useTranslation } from 'react-i18next';
import { WiFiCard } from '../components/cards/WiFiCard';
import { WifiChannelGraph } from '../components/cards/WiFiChannelGraph';
import { RequireFeature } from '../components/ui/RequireFeature';
import { WiFiAirspaceCard } from '../components/wifi/WiFiAirspaceCard';
import { WiFiAnomaliesCard } from '../components/wifi/WiFiAnomaliesCard';
import { useAppContext } from '../contexts/AppContext';
import { CardAbsent, CardGrid } from '../ui/CardGrid';

/**
 * Wi-Fi — Card grid.
 *
 * No rollup: each card carries its own state, and a band above them would
 * only restate whichever one is loudest.
 *
 * Two kinds of conditional membership meet here. A wired interface cannot
 * produce any of it, which the page says once rather than five times. The
 * tier-gated cards are absent for a different reason — the licence, not the
 * hardware — and say so, because "this needs a tier you do not have" and
 * "this needs a radio you do not have" have different fixes.
 */
export function WifiPage() {
  const { t } = useTranslation('pages');
  const { cards, loading, isWifi, channelGraphData, channelGraphLoading } = useAppContext();

  /* Not one absent card among others — the whole page is inapplicable, so
     the note is the page rather than a lone tile in a four-column grid. */
  if (!isWifi) {
    return (
      <CardAbsent id="wireless-data" label={t('wifi.wiredLabel')} reason={t('wifi.wiredReason')} />
    );
  }

  return (
    <CardGrid>
      <WiFiCard data={cards.wifi} loading={loading} visible={true} />
      <WifiChannelGraph data={channelGraphData} loading={channelGraphLoading} visible={true} />

      {/* Wi-Fi visibility (W5/W6, #2351): live airspace tree + anomaly stream
          from scan results, and from 802.11 management-frame capture when a
          monitor-capable interface feeds it (internal/wifi/visibility). Both
          cards are Starter-gated; the clients in the tree are Pro. */}
      <RequireFeature
        feature="wifi_analysis"
        fallback={
          <CardAbsent id="airspace" label={t('wifi.airspaceLabel')} reason={t('wifi.tierHint')} />
        }
      >
        <WiFiAirspaceCard />
      </RequireFeature>

      <RequireFeature
        feature="wifi_analysis"
        fallback={
          <CardAbsent
            id="association-anomalies"
            label={t('wifi.anomaliesLabel')}
            reason={t('wifi.tierHint')}
          />
        }
      >
        <WiFiAnomaliesCard />
      </RequireFeature>
    </CardGrid>
  );
}
