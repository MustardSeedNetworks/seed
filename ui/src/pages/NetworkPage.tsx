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
        iconColorClass="text-module-sap"
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
