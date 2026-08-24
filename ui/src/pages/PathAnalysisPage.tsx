import { Trans } from 'react-i18next';
import { NetworkDiscoveryCard } from '../components/cards/NetworkDiscoveryCard';
import { PathDiscoveryCard } from '../components/cards/PathDiscoveryCard';
import { RequireFeature } from '../components/ui/RequireFeature';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';

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
        <div className="rounded-lg border border-status-warning/30 bg-status-warning/5 pad text-sm text-status-warning">
          <Trans
            i18nKey="path.proGate"
            ns="pages"
            components={{
              code: <code className="mx-1 px-1 rounded bg-surface-raised" />,
              code2: <code className="ml-tight px-1 rounded bg-surface-raised" />,
            }}
          />
        </div>
      }
    >
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
    </RequireFeature>
  );
}
