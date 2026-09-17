import { useTranslation } from 'react-i18next';
import { BonjourCard } from '../components/cards/BonjourCard';
import { DnsCard } from '../components/cards/DnsCard';
import { GatewayCard } from '../components/cards/GatewayCard';
import { NeighbourCacheCard } from '../components/cards/NeighbourCacheCard';
import { NetworkCard } from '../components/cards/NetworkCard';
import { NetworkDiscoveryCard } from '../components/cards/NetworkDiscoveryCard';
import { PublicIpCard } from '../components/cards/PublicIpCard';
import { SwitchCard } from '../components/cards/SwitchCard';
import { useAppContext } from '../contexts/AppContext';
import { CardGrid, CardSlot } from '../ui/CardGrid';
import { StatusRollup } from '../ui/StatusRollup';
import { networkRollup } from './networkRollup';

/**
 * Network — Card grid, with the rollup it already had.
 *
 * Converted with the archetype so the one card that is conditional here says
 * why it is missing: switch and VLAN discovery reads the wire, and a wireless
 * interface has no port to ask.
 */
export function NetworkPage() {
  const { t } = useTranslation('pages');
  const {
    cards,
    loading,
    isWifi,
    displayOptions,
    cardSettings,
    networkDiscovery,
    scanError,
    triggerDeviceScan,
    openSettings,
  } = useAppContext();

  /* Overview opens with the rollup. The question this page answers is whether
     the upstream link is healthy, and five cards each reporting their own
     state make the reader assemble that themselves.

     The derivation lives in networkRollup so it can be read as a table of
     states rather than a nested ternary, and so the rule the band gets wrong
     when it drifts — a card is present, therefore all is well — is pinned by
     unit tests rather than by an E2E that would have to break the link. */
  const rollup = networkRollup({ gateway: cards.gateway, dns: cards.dns, loading });

  return (
    <>
      <StatusRollup
        state={rollup.state}
        headline={t(rollup.headlineKey)}
        body={rollup.bodyKey ? t(rollup.bodyKey) : undefined}
        figures={[
          { label: t('network.figureGateway'), value: t(rollup.gatewayFigureKey) },
          { label: t('network.figureDns'), value: t(rollup.dnsFigureKey) },
        ]}
      />

      <CardGrid>
        {/* On a wireless interface these wait for the Wi-Fi payload rather
            than rendering four cards about an interface not yet identified.
            That absence is transient, so it is quiet. */}
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
            <NeighbourCacheCard />
            <BonjourCard />
            <PublicIpCard data={cards.publicip} loading={loading} />
          </>
        )}
        {/* What discovery found on this segment. Until #2674 the list had no
            home on a routed page at all — its only mount was Path Analysis,
            behind the Pro feature gate — so a fresh install that had scanned
            correctly still showed the operator nothing here. */}
        <NetworkDiscoveryCard
          data={networkDiscovery}
          loading={loading}
          scanError={scanError}
          onScan={triggerDeviceScan}
          discoveryEnabled={cardSettings.networkDiscovery.enabled}
          onOpenSettings={openSettings}
        />
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
      </CardGrid>
    </>
  );
}
