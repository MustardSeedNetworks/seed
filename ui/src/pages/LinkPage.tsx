import type { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';
import { CableCard } from '../components/cards/CableCard';
import { DriverStatsCard } from '../components/cards/DriverStatsCard';
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
  const { t } = useTranslation('pages');
  const { cards, loading, isWifi, displayOptions } = useAppContext();
  const rollup = describeLink({ link: cards.link, loading, isWifi, t });

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
            id: 'wired-link',
            label: t('link.wiredAbsentLabel'),
            reason: t('link.wiredAbsentReason'),
          }}
        >
          <LinkCard data={cards.link} loading={loading} />
        </CardSlot>

        {/* Cable diagnostics run when the link is down. Their absence on a
            healthy link is not a missing measurement, so it is quiet. */}
        <CardSlot present={!isWifi && cards.link?.linkUp === false} absence="quiet">
          <CableCard data={cards.cable} loading={loading} unitSystem={displayOptions.unitSystem} />
        </CardSlot>

        {/* Driver counters apply to whichever interface is selected, wired or
            wireless, so this sits at grid level rather than in either slot. It
            gates itself: on a platform without ethtool the card explains why
            instead of showing an empty table. */}
        <DriverStatsCard />
      </CardGrid>
    </>
  );
}

interface LinkRollupInput {
  link: LinkData | null;
  loading: boolean;
  isWifi: boolean;
  t: TFunction<'pages'>;
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
function describeLink({ link, loading, isWifi, t }: LinkRollupInput): LinkRollup {
  if (isWifi) {
    return {
      state: 'unknown',
      headline: t('link.rollupWireless'),
      body: t('link.rollupWirelessBody'),
      figures: [],
    };
  }
  if (loading) {
    return { state: 'unknown', headline: t('link.rollupReading'), figures: [] };
  }
  if (!link) {
    return {
      state: 'unknown',
      headline: t('link.rollupNoData'),
      body: t('link.rollupNoDataBody'),
      figures: [],
    };
  }

  const figures: RollupFigure[] = [
    { label: t('link.figureSpeed'), value: link.speed || '—' },
    { label: t('link.figureDuplex'), value: link.duplex || '—' },
    { label: t('link.figureMtu'), value: link.mtu === undefined ? '—' : String(link.mtu) },
    {
      label: t('link.figureFlaps'),
      value: link.flapCount24h === undefined ? '—' : String(link.flapCount24h),
    },
  ];

  if (!link.carrier) {
    return {
      state: 'crit',
      headline: t('link.rollupNoCarrier'),
      body: t('link.rollupNoCarrierBody'),
      figures,
    };
  }
  if (!link.linkUp) {
    return {
      state: 'crit',
      headline: t('link.rollupDown'),
      body: t('link.rollupDownBody'),
      figures,
    };
  }
  if (!link.hasIp) {
    return {
      state: 'warn',
      headline: t('link.rollupNoAddress'),
      body: t('link.rollupNoAddressBody'),
      figures,
    };
  }
  if (link.duplex && link.duplex.toLowerCase() === 'half') {
    return {
      state: 'warn',
      headline: t('link.rollupHalfDuplex'),
      body: t('link.rollupHalfDuplexBody'),
      figures,
    };
  }
  return {
    state: 'ok',
    headline: t('link.rollupUp', { speed: link.speed || t('link.rollupUnreportedSpeed') }),
    figures,
  };
}
