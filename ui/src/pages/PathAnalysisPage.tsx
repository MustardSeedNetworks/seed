import { MultiPathCard } from '../components/cards/MultiPathCard';
import { NetworkDiscoveryCard } from '../components/cards/NetworkDiscoveryCard';
import { PathDiscoveryCard } from '../components/cards/PathDiscoveryCard';
import { PathMonitorCard } from '../components/cards/PathMonitorCard';
import { PathMTUCard } from '../components/cards/PathMTUCard';
import { PathAnalysisPreview } from '../components/previews/PathAnalysisPreview';
import { GatedPreview } from '../components/ui/GatedPreview';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';

export function PathAnalysisPage() {
  const {
    cards,
    loading,
    isWifi,
    cardSettings,
    networkDiscovery,
    scanError,
    triggerDeviceScan,
    registerTraceHopHandler,
    registerPathMonitorHandler,
    openSettings,
  } = useAppContext();

  return (
    <GatedPreview feature="path_analysis" preview={<PathAnalysisPreview />}>
      <div className={layout.grid.cards}>
        {(!isWifi || cards.wifi) && (
          <>
            <PathDiscoveryCard
              gateway={cards.gateway?.gateway}
              dnsServer={cards.dns?.servers?.[0] ?? cards.dns?.server}
              onRegisterTraceHandler={registerTraceHopHandler}
            />
            <PathMonitorCard
              defaultTarget={cards.gateway?.gateway ?? ''}
              linkDown={!isWifi && cards.link?.linkUp === false}
              subscribe={registerPathMonitorHandler}
            />
            <MultiPathCard />
            <PathMTUCard />
          </>
        )}
        {/* The card renders with discovery switched off as well: hiding it was
            half of #2674 — nothing found and no page saying why. */}
        {!isWifi && (
          <NetworkDiscoveryCard
            data={networkDiscovery}
            loading={loading}
            scanError={scanError}
            onScan={triggerDeviceScan}
            discoveryEnabled={cardSettings.networkDiscovery.enabled}
            onOpenSettings={openSettings}
          />
        )}
      </div>
    </GatedPreview>
  );
}
