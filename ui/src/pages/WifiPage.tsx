import { Wifi } from 'lucide-react';
import { WiFiCard } from '../components/cards/WiFiCard';
import { WifiChannelGraph } from '../components/cards/WiFiChannelGraph';
import { WiFiSurveyCard } from '../components/cards/WiFiSurveyCard';
import { BetaBadge } from '../components/ui/BetaBadge';
import { Card } from '../components/ui/card';
import { RequireFeature } from '../components/ui/RequireFeature';
import { WiFiAirspaceCard } from '../components/wifi/WiFiAirspaceCard';
import { WiFiAnomaliesCard } from '../components/wifi/WiFiAnomaliesCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function WifiPage() {
  const { cards, loading, isWifi, currentInterface, channelGraphData, channelGraphLoading } =
    useAppContext();

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={Wifi}
        title="Wi-Fi"
        description="Wireless link, channel survey, and channel-overlap visualisation."
        iconColorClass="text-module-wifi"
      />
      <div className={layout.grid.cards}>
        {isWifi ? <WiFiCard data={cards.wifi} loading={loading} visible={true} /> : null}
        {isWifi ? <WiFiSurveyCard isWifi={isWifi} currentInterface={currentInterface} /> : null}
        {isWifi ? (
          <WifiChannelGraph
            data={channelGraphData}
            loading={channelGraphLoading}
            visible={isWifi}
          />
        ) : null}

        {/* Wi-Fi visibility (W5/W6): live airspace tree + anomaly stream from
            802.11 management-frame capture (internal/wifi/visibility). Each card
            is Pro-gated and degrades to an empty/last-observed view when no
            monitor-capable interface is feeding the capture loop. */}
        <RequireFeature feature="wifi_management_capture">
          <WiFiAirspaceCard />
        </RequireFeature>

        <RequireFeature feature="wifi_association_forensics">
          <WiFiAnomaliesCard />
        </RequireFeature>

        {/* Phase 2.5 scaffolding — fills with real data when per-client roam
            correlation lands. See
            msn-docs-internal/01-Strategy/SEED_NMS_EXPANSION.md. */}
        <RequireFeature feature="wifi_roam_analysis">
          <Card
            title="Roam Analysis"
            subtitle="Disassoc/(re)assoc correlation per client MAC with 802.11r FT detection."
            status="unknown"
            headerAction={<BetaBadge />}
          >
            <p data-testid="wifi-roam-analysis-pending" className="text-sm text-text-muted">
              Capture lands in Seed v1.0 — see release notes when Phase 2.5 ships.
            </p>
          </Card>
        </RequireFeature>

        {!isWifi && (
          <div data-testid="wifi-wired-fallback" className="col-span-full text-sm text-text-muted">
            Switch to Wi-Fi mode from the header to view wireless data.
          </div>
        )}
      </div>
    </section>
  );
}
