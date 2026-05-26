import { Route } from 'lucide-react';
import { NetworkDiscoveryCard } from '../components/cards/NetworkDiscoveryCard';
import { PathDiscoveryCard } from '../components/cards/PathDiscoveryCard';
import { RequireFeature } from '../components/ui/RequireFeature';
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
    <RequireFeature
      feature="path_analysis"
      fallback={
        <section className="space-y-6">
          <Breadcrumbs />
          <PageHeader
            icon={Route}
            title="Path Analysis"
            description="L2/L3 path discovery, traceroute hops, and on-link device discovery."
            iconColorClass="text-module-roots"
          />
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 text-sm text-amber-200">
            Path Analysis is a Pro-tier feature. Start a 14-day Pro trial with
            <code className="mx-1 px-1 rounded bg-surface-raised">seed license trial</code>
            or activate a Pro key with
            <code className="ml-1 px-1 rounded bg-surface-raised">
              seed license activate -k &lt;KEY&gt;
            </code>
            .
          </div>
        </section>
      }
    >
      <section className="space-y-6">
        <Breadcrumbs />
        <PageHeader
          icon={Route}
          title="Path Analysis"
          description="L2/L3 path discovery, traceroute hops, and on-link device discovery."
          iconColorClass="text-module-roots"
        />
        <div className={layout.grid.cards}>
          {(!isWifi || cards.wifi) && (
            <PathDiscoveryCard
              gateway={cards.gateway?.gateway}
              dnsServer={cards.dns?.servers?.[0] ?? cards.dns?.server}
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
    </RequireFeature>
  );
}
