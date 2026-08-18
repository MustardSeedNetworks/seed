import { BluetoothCard } from '../components/cards/BluetoothCard';
import { GuestNetworkAuditCard } from '../components/cards/GuestNetworkAuditCard';
import { MfaCard } from '../components/cards/MfaCard';
import { layout } from '../styles/theme';

export function SecurityPage() {
  return (
    <div className={layout.grid.cards}>
      <MfaCard />
      <GuestNetworkAuditCard />
      <BluetoothCard />
    </div>
  );
}
