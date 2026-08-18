import { DnsCard } from '../components/cards/DnsCard';
import { GatewayCard } from '../components/cards/GatewayCard';
import { NetworkCard } from '../components/cards/NetworkCard';
import { PublicIpCard } from '../components/cards/PublicIpCard';
import { SwitchCard } from '../components/cards/SwitchCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';
import { type RollupState, StatusRollup } from '../ui/StatusRollup';

export function NetworkPage() {
  const { cards, loading, isWifi, displayOptions } = useAppContext();

  /* Overview opens with the rollup. The question this page answers is whether
     the upstream link is healthy, and five cards each reporting their own
     state make the reader assemble that themselves.

     While the probes are still running the answer is not "healthy", it is "not
     known yet" — the difference matters because a card with no data and a card
     that failed look the same to a reader in a hurry. */
  const gatewayUp = Boolean(cards.gateway);
  const dnsUp = Boolean(cards.dns);
  const state: RollupState = loading ? 'unknown' : !gatewayUp ? 'crit' : !dnsUp ? 'warn' : 'ok';

  const headline = loading
    ? 'Probing the upstream link'
    : !gatewayUp
      ? 'No default gateway was found on this interface'
      : !dnsUp
        ? 'A gateway is present but no DNS resolver answered'
        : 'The upstream link is answering';

  const body = loading
    ? undefined
    : !gatewayUp
      ? 'Without a gateway nothing beyond this segment is reachable. Check the interface selection and the DHCP lease below.'
      : !dnsUp
        ? 'Names will not resolve even though routing works. The resolver list is in the DNS card below.'
        : undefined;

  return (
    <>
      <StatusRollup
        state={state}
        headline={headline}
        body={body}
        figures={[
          { label: 'Gateway', value: gatewayUp ? 'Up' : 'None' },
          { label: 'DNS', value: dnsUp ? 'Up' : 'None' },
        ]}
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
    </>
  );
}
