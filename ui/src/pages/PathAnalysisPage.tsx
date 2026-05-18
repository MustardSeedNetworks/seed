import { Route } from 'lucide-react';
import { NetworkDiscoveryCard } from '../components/cards/NetworkDiscoveryCard';
import { PathDiscoveryCard } from '../components/cards/PathDiscoveryCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function PathAnalysisPage() {
  const {
    cards,
    loading,
    isWifi,
    cardSettings,
    networkDiscovery,
    triggerDeviceScan,
    registerTraceHopHandler,
  } = useAppContext();

  return (
    <section class="space-y-6">
      <Breadcrumbs />
      <PageHeader
        icon={Route}
        title="Path Analysis"
        description="L2/L3 path discovery, traceroute hops, and on-link device discovery."
        iconColorClass="text-module-roots"
      />
      <div class={layout.grid.cards}>
        {(!isWifi || cards.wifi) && (
          <PathDiscoveryCard
            gateway={cards.gateway?.gateway}
            dnsServer={cards.dns?.servers?.[0]?.address}
            onRegisterTraceHandler={registerTraceHopHandler}
          />
        )}
        {!isWifi && cardSettings.networkDiscovery.enabled && (
          <NetworkDiscoveryCard
            data={networkDiscovery}
            loading={loading}
            onScan={triggerDeviceScan}
          />
        )}
      </div>
    </section>
  );
}
