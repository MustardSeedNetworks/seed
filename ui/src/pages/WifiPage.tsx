import { Wifi } from 'lucide-react';
import { WiFiCard } from '../components/cards/WiFiCard';
import { WifiChannelGraph } from '../components/cards/WiFiChannelGraph';
import { WiFiSurveyCard } from '../components/cards/WiFiSurveyCard';
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
        iconColorClass="text-module-canopy"
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
        {!isWifi && (
          <div className="col-span-full text-sm text-text-muted">
            Switch to Wi-Fi mode from the header to view wireless data.
          </div>
        )}
      </div>
    </section>
  );
}
