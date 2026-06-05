import { Shield } from 'lucide-react';
import { BluetoothCard } from '../components/cards/BluetoothCard';
import { GuestNetworkAuditCard } from '../components/cards/GuestNetworkAuditCard';
import { MfaCard } from '../components/cards/MfaCard';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function SecurityPage() {
  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={Shield}
        title="Security"
        description="Guest network isolation audit and security posture checks."
        iconColorClass="text-module-security"
      />
      <div className={layout.grid.cards}>
        <MfaCard />
        <GuestNetworkAuditCard />
        <BluetoothCard />
      </div>
    </section>
  );
}
