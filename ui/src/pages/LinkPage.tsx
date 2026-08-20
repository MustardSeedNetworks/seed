import { CableCard } from '../components/cards/CableCard';
import { LinkCard, type LinkData } from '../components/cards/LinkCard';
import { WiFiCard } from '../components/cards/WiFiCard';
import { useAppContext } from '../contexts/AppContext';
import { CardGrid, CardSlot } from '../ui/CardGrid';
import { type RollupFigure, type RollupState, StatusRollup } from '../ui/StatusRollup';

/**
 * Link — Card grid.
 *
 * The one Card grid page that opens with a rollup, because it is the one that
 * can be wrong on its own: no carrier is a real state, and a grid of cards
 * reporting it card-by-card buries the sentence someone needs first.
 *
 * On a wireless interface the wired cards say why they are not here. That is
 * the archetype's rule rather than this page's habit — a card the subject
 * cannot produce explains itself, and one that is simply not needed (cable
 * diagnostics on a healthy link) stays quiet.
 */
export function LinkPage() {
  const { cards, loading, isWifi, displayOptions } = useAppContext();
  const rollup = describeLink({ link: cards.link, loading, isWifi });

  return (
    <>
      <StatusRollup
        state={rollup.state}
        headline={rollup.headline}
        body={rollup.body}
        figures={rollup.figures}
      />

      <CardGrid>
        {/* On a wireless interface the Wi-Fi card is the link. Its absence on
            a wired one needs no note: the wired cards are right there. */}
        <CardSlot present={isWifi} absence="quiet">
          <WiFiCard data={cards.wifi} loading={loading} visible={true} />
        </CardSlot>

        <CardSlot
          present={!isWifi}
          absence={{
            label: 'Wired link',
            reason:
              'This interface is wireless — carrier, duplex and cable diagnostics come from a wired port.',
          }}
        >
          <LinkCard data={cards.link} loading={loading} />
        </CardSlot>

        {/* Cable diagnostics run when the link is down. Their absence on a
            healthy link is not a missing measurement, so it is quiet. */}
        <CardSlot present={!isWifi && cards.link?.linkUp === false} absence="quiet">
          <CableCard data={cards.cable} loading={loading} unitSystem={displayOptions.unitSystem} />
        </CardSlot>
      </CardGrid>
    </>
  );
}

interface LinkRollupInput {
  link: LinkData | null;
  loading: boolean;
  isWifi: boolean;
}

interface LinkRollup {
  state: RollupState;
  headline: string;
  body?: string;
  figures: RollupFigure[];
}

/**
 * describeLink turns link state into the sentence the page leads with.
 *
 * Loading is `unknown` rather than `ok`, and so is a missing payload: a page
 * that reads "all clear" while nothing has arrived is the failure the rollup
 * exists to prevent.
 */
function describeLink({ link, loading, isWifi }: LinkRollupInput): LinkRollup {
  if (isWifi) {
    return {
      state: 'unknown',
      headline: 'This interface is wireless',
      body: 'Signal, SSID and channel are on the Wi-Fi page; carrier state comes from a wired port.',
      figures: [],
    };
  }
  if (loading) {
    return { state: 'unknown', headline: 'Reading the interface', figures: [] };
  }
  if (!link) {
    return {
      state: 'unknown',
      headline: 'Link data is not arriving',
      body: 'The interface has reported nothing. Nothing below is current.',
      figures: [],
    };
  }

  const figures: RollupFigure[] = [
    { label: 'Speed', value: link.speed || '—' },
    { label: 'Duplex', value: link.duplex || '—' },
    { label: 'MTU', value: link.mtu === undefined ? '—' : String(link.mtu) },
    {
      label: 'Flaps 24h',
      value: link.flapCount24h === undefined ? '—' : String(link.flapCount24h),
    },
  ];

  if (!link.carrier) {
    return {
      state: 'crit',
      headline: 'No carrier on this interface',
      body: 'Nothing is detected on the wire. Check the cable and the far-end port; the cable test below reports where the fault is.',
      figures,
    };
  }
  if (!link.linkUp) {
    return {
      state: 'crit',
      headline: 'The link is down',
      body: 'Carrier is present but the interface is not up. Check whether it is administratively disabled.',
      figures,
    };
  }
  if (!link.hasIp) {
    return {
      state: 'warn',
      headline: 'The link is up with no routable address',
      body: 'Layer 2 is healthy and layer 3 is not. DHCP or the static configuration is the next thing to check.',
      figures,
    };
  }
  if (link.duplex && link.duplex.toLowerCase() === 'half') {
    return {
      state: 'warn',
      headline: 'The link negotiated half duplex',
      body: 'Half duplex on a switched port is almost always a negotiation mismatch, and it costs throughput.',
      figures,
    };
  }
  return {
    state: 'ok',
    headline: `The link is up at ${link.speed || 'an unreported speed'}`,
    figures,
  };
}
