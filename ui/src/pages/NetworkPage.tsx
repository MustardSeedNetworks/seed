import { Server } from 'lucide-react';
import { DnsCard } from '../components/cards/DnsCard';
import { GatewayCard } from '../components/cards/GatewayCard';
import { NetworkCard } from '../components/cards/NetworkCard';
import { PublicIpCard } from '../components/cards/PublicIpCard';
import { SwitchCard } from '../components/cards/SwitchCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function NetworkPage() {
  const { cards, loading, isWifi, displayOptions } = useAppContext();

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={Server}
        title="Network"
        description="DHCP, gateway, DNS, public IP, and upstream switch detection."
        iconColorClass="text-module-telemetry"
        help={
          <div className="stack-md">
            <p>
              The Network page shows the diagnostic state of the upstream link: DHCP lease, default
              gateway, DNS resolvers, the public IP the gateway uses, and (when wired) the
              directly-attached switch and its VLAN configuration.
            </p>
            <p>
              Each card refreshes on its own schedule and reflects the active interface selected in
              the header. If a card shows "no data", either the interface has not yet been probed,
              or that piece of the upstream config is not present (e.g., no IPv6 gateway).
            </p>
          </div>
        }
      />
      <div className={layout.grid.cards}>
        {(!isWifi || cards.wifi) && (
          <>
            <NetworkCard
              data={cards.dhcp}
              publicIp={cards.publicip}
              loading={loading}
              showPublicIp={displayOptions.showPublicIp}
            />
            <GatewayCard data={cards.gateway} loading={loading} />
            <DnsCard data={cards.dns} loading={loading} />
            <PublicIpCard data={cards.publicip} loading={loading} />
          </>
        )}
        {!isWifi && <SwitchCard data={cards.switch} vlanData={cards.vlan} loading={loading} />}
      </div>
    </section>
  );
}
