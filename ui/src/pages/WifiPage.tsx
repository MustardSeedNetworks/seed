import { useTranslation } from 'react-i18next';
import { WiFiCard } from '../components/cards/WiFiCard';
import { WifiChannelGraph } from '../components/cards/WiFiChannelGraph';
import { BetaBadge } from '../components/ui/BetaBadge';
import { Card } from '../components/ui/card';
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
 * Pro-gated cards are absent for a different reason — the licence, not the
 * hardware — and say so, because "this needs a tier you do not have" and
 * "this needs a radio you do not have" have different fixes.
 */
const TIER_HINT = 'Available on Seed Pro. Run `seed license trial` for a 14-day trial.';

export function WifiPage() {
  const { t } = useTranslation('pages');
  const { cards, loading, isWifi, channelGraphData, channelGraphLoading } = useAppContext();

  /* Not one absent card among others — the whole page is inapplicable, so
     the note is the page rather than a lone tile in a four-column grid. */
  if (!isWifi) {
    return (
      <CardAbsent
        label="Wireless data"
        reason="This interface is wired. Switch to a Wi-Fi interface from the header to see signal, channels and airspace."
      />
    );
  }

  return (
    <CardGrid>
      <WiFiCard data={cards.wifi} loading={loading} visible={true} />
      <WifiChannelGraph data={channelGraphData} loading={channelGraphLoading} visible={true} />

      {/* Wi-Fi visibility (W5/W6): live airspace tree + anomaly stream from
          802.11 management-frame capture (internal/wifi/visibility). Each card
          is Pro-gated and degrades to an empty/last-observed view when no
          monitor-capable interface is feeding the capture loop. */}
      <RequireFeature
        feature="wifi_management_capture"
        fallback={<CardAbsent label="Airspace" reason={TIER_HINT} />}
      >
        <WiFiAirspaceCard />
      </RequireFeature>

      <RequireFeature
        feature="wifi_association_forensics"
        fallback={<CardAbsent label="Association anomalies" reason={TIER_HINT} />}
      >
        <WiFiAnomaliesCard />
      </RequireFeature>

      {/* Phase 2.5 scaffolding — fills with real data when per-client roam
          correlation lands. See
          msn-docs-internal/01-Strategy/SEED_NMS_EXPANSION.md. */}
      <RequireFeature
        feature="wifi_roam_analysis"
        fallback={<CardAbsent label="Roam analysis" reason={TIER_HINT} />}
      >
        <Card
          title="Roam Analysis"
          subtitle="Disassoc/(re)assoc correlation per client MAC with 802.11r FT detection."
          status="unknown"
          headerAction={<BetaBadge />}
        >
          <p data-testid="wifi-roam-analysis-pending" className="text-sm text-text-muted">
            {t('wifi.capturePlanned')}
          </p>
        </Card>
      </RequireFeature>
    </CardGrid>
  );
}
