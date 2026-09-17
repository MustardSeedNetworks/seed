import { BluetoothCard } from '../components/cards/BluetoothCard';
import { GuestNetworkAuditCard } from '../components/cards/GuestNetworkAuditCard';
import { InsecurePortScanCard } from '../components/cards/InsecurePortScanCard';
import { MfaCard } from '../components/cards/MfaCard';
import { NetworkDiscoveryCard } from '../components/cards/NetworkDiscoveryCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';

export function SecurityPage() {
  const { loading, cardSettings, networkDiscovery, scanError, triggerDeviceScan, openSettings } =
    useAppContext();

  return (
    <div className={layout.grid.cards}>
      {/* The posture cards below assess what discovery found, so the inventory
          leads (#2674). It also carries the reason the page can be empty:
          discovery off, still sweeping, or a genuinely quiet segment. */}
      <NetworkDiscoveryCard
        data={networkDiscovery}
        loading={loading}
        scanError={scanError}
        onScan={triggerDeviceScan}
        discoveryEnabled={cardSettings.networkDiscovery.enabled}
        onOpenSettings={openSettings}
      />
      <MfaCard />
      <GuestNetworkAuditCard />
      <InsecurePortScanCard />
      <BluetoothCard />
    </div>
  );
}
