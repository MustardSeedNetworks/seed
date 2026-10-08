/**
 * One dashboard widget: an existing card, fed from the same live state its
 * own page uses, so a widget never disagrees with the page it came from.
 */
import type { TFunction } from 'i18next';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { DnsCard } from '../../components/cards/DnsCard';
import { DriverStatsCard } from '../../components/cards/DriverStatsCard';
import { GatewayCard } from '../../components/cards/GatewayCard';
import { LinkCard } from '../../components/cards/LinkCard';
import { NeighbourCacheCard } from '../../components/cards/NeighbourCacheCard';
import { NetworkCard } from '../../components/cards/NetworkCard';
import { PublicIpCard } from '../../components/cards/PublicIpCard';
import { SwitchCard } from '../../components/cards/SwitchCard';
import { WiFiCard } from '../../components/cards/WiFiCard';
import { useAppContext } from '../../contexts/AppContext';
import { CardSlot } from '../../ui/CardGrid';
import type { DashboardWidgetId } from './layout';

/** The name a widget goes by in the customize list. */
export function widgetLabel(t: TFunction<'pages'>, id: DashboardWidgetId): string {
  switch (id) {
    case 'link':
      return t('dashboard.widget.link');
    case 'network':
      return t('dashboard.widget.network');
    case 'gateway':
      return t('dashboard.widget.gateway');
    case 'dns':
      return t('dashboard.widget.dns');
    case 'publicIp':
      return t('dashboard.widget.publicIp');
    case 'switch':
      return t('dashboard.widget.switch');
    case 'neighbours':
      return t('dashboard.widget.neighbours');
    case 'driverStats':
      return t('dashboard.widget.driverStats');
  }
}

export function DashboardWidget({ id }: { id: DashboardWidgetId }): JSX.Element {
  const { t } = useTranslation('pages');
  const { cards, loading, isWifi, displayOptions } = useAppContext();

  switch (id) {
    case 'link':
      // On a wireless interface the Wi-Fi card is the link, as on /link.
      return isWifi ? (
        <WiFiCard data={cards.wifi} loading={loading} visible={true} />
      ) : (
        <LinkCard data={cards.link} loading={loading} />
      );
    case 'network':
      return (
        <NetworkCard
          data={cards.dhcp}
          publicIp={cards.publicip}
          loading={loading}
          showPublicIp={displayOptions.showPublicIp}
        />
      );
    case 'gateway':
      return <GatewayCard data={cards.gateway} loading={loading} />;
    case 'dns':
      return <DnsCard data={cards.dns} loading={loading} />;
    case 'publicIp':
      return <PublicIpCard data={cards.publicip} loading={loading} />;
    case 'switch':
      return (
        <CardSlot
          present={!isWifi}
          absence={{
            id: 'switch-and-vlan',
            label: t('network.switchAbsentLabel'),
            reason: t('network.switchAbsentReason'),
          }}
        >
          <SwitchCard data={cards.switch} vlanData={cards.vlan} loading={loading} />
        </CardSlot>
      );
    case 'neighbours':
      return <NeighbourCacheCard />;
    case 'driverStats':
      return <DriverStatsCard />;
  }
}
