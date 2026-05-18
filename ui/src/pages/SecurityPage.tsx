import { Shield } from 'lucide-react';
import { GuestNetworkAuditCard } from '../components/cards/GuestNetworkAuditCard';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function SecurityPage() {
  return (
    <section class="space-y-6">
      <Breadcrumbs />
      <PageHeader
        icon={Shield}
        title="Security"
        description="Guest network isolation audit and security posture checks."
        iconColorClass="text-module-shell"
      />
      <div class={layout.grid.cards}>
        <GuestNetworkAuditCard />
      </div>
    </section>
  );
}
